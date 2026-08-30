// seeder 数据灌入工具：按计算光刻领域知识生成 model/case/layout/layer 四个集合的
// 海量示例数据（真实文件树 + 数据项 + 关联关系），用于演示与联调。
//
// 用法：
//
//	. .\scripts\goenv.ps1
//	$env:DATACENTER_DATABASE_URI='mongodb://localhost:27017'
//	$env:DATACENTER_DATABASE_NAME='datacenter'
//	$env:DATACENTER_DATA_ROOT_DIR='D:\gocode\deepseek\datacenter\.nfsdata'
//	go run ./cmd/seeder -config config/config.yaml
//
// 参数：
//
//	-fresh    先清空同名集合再重建（默认 true，保证可重复执行）
//	-models   模型数量（默认 3000）
//	-layouts  版图数量（默认 2000）
//	-layers   图层数量（默认 0 = 每个版图随机 4~14 层）
//	-cases    测试用例数量（默认 3000）
//	-workers  文件创建并发数（默认 16）
//	-no-files 跳过真实文件创建（仅写元数据，默认 false）
//	-seed     随机种子（默认按当前时间，固定值可复现）
package main

import (
	"context"
	"flag"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"datacenter/internal/config"
	"datacenter/internal/database"
	"datacenter/internal/logger"
	"datacenter/internal/model"
	"datacenter/internal/service"
	"datacenter/internal/store"
)

// ---------- 计算光刻领域常量 ----------

// 工艺节点（nm），按先进制程分布加权的取值池
var nodes = []string{"3", "5", "7", "10", "14", "16", "22", "28", "32", "45", "65"}
var nodeWeights = []int{1, 2, 4, 4, 3, 2, 2, 2, 1, 1, 1}

// 各节点典型最小节距（nm），用于图层 CD 约束生成
var pitchByNode = map[string]float64{
	"3": 24, "5": 28, "7": 36, "10": 44, "14": 56, "16": 64,
	"22": 80, "28": 100, "32": 112, "45": 140, "65": 180,
}

// 各节点典型数值孔径
var naByNode = map[string]float64{
	"3": 1.45, "5": 1.35, "7": 1.35, "10": 1.35, "14": 1.35, "16": 1.2,
	"22": 1.2, "28": 0.93, "32": 0.93, "45": 0.85, "65": 0.75,
}

var layerPool = []string{
	"M1", "M2", "M3", "M4", "M5", "M6", "M7", "M8", "M9", "M10", "M11", "M12",
	"V1", "V2", "V3", "V4", "V5", "V6", "V7", "V8", "V9",
	"OD", "POLY", "NW", "PW", "CT", "AP",
}

var owners = []string{
	"zhang.wei", "li.na", "wang.fang", "liu.qiang", "chen.jing",
	"yang.lei", "huang.xin", "zhao.min", "zhou.yu", "wu.han",
}

// ---------- 工具 ----------

func round2(f float64) float64 { return math.Round(f*100) / 100 }

func pickWeighted(r *rand.Rand, items []string, weights []int) string {
	total := 0
	for _, w := range weights {
		total += w
	}
	n := r.Intn(total)
	for i, w := range weights {
		if n < w {
			return items[i]
		}
		n -= w
	}
	return items[len(items)-1]
}

func pick(r *rand.Rand, items []string) string { return items[r.Intn(len(items))] }

func pickN(r *rand.Rand, items []string, n int) []string {
	pool := append([]string{}, items...)
	r.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })
	if n > len(pool) {
		n = len(pool)
	}
	return pool[:n]
}

func pickKw(r *rand.Rand, pool []string) []interface{} {
	n := 1 + r.Intn(3)
	if n > len(pool) {
		n = len(pool)
	}
	out := make([]interface{}, 0, n)
	for _, k := range pickN(r, pool, n) {
		out = append(out, k)
	}
	return out
}

func randDate(r *rand.Rand, maxDaysAgo int) time.Time {
	return time.Now().Add(-time.Duration(r.Intn(maxDaysAgo*24*60)) * time.Minute).UTC()
}

// layerCounts 计算每个版图的图层数：total<=0 时每版图随机 4~14 层，否则均匀分配
func layerCounts(r *rand.Rand, nLayouts, total int) []int {
	out := make([]int, nLayouts)
	if total <= 0 {
		for i := range out {
			out[i] = 4 + r.Intn(11)
		}
		return out
	}
	base, rem := total/nLayouts, total%nLayouts
	for i := range out {
		out[i] = base
	}
	for i := 0; i < rem; i++ {
		out[i]++
	}
	return out
}

func layerTypeOf(name string) string {
	switch {
	case strings.HasPrefix(name, "M"):
		return "metal"
	case strings.HasPrefix(name, "V"):
		return "via"
	case name == "OD":
		return "active"
	case name == "POLY":
		return "poly"
	case name == "NW" || name == "PW":
		return "well"
	case name == "CT":
		return "cut"
	case name == "AP":
		return "seal"
	}
	return "metal"
}

// ---------- 标签定义（计算光刻领域） ----------

func modelSchema() []model.TagDefinition {
	return []model.TagDefinition{
		{Name: "name", Type: model.TagTypeString, Required: true},
		{Name: "node", Type: model.TagTypeEnum, Required: true, EnumValues: nodes},
		{Name: "model_type", Type: model.TagTypeEnum, Required: true,
			EnumValues: []string{"optical", "resist", "etch", "full"}},
		{Name: "lib_type", Type: model.TagTypeEnum, EnumValues: []string{"im", "abbe", "socs", "r3d"}},
		{Name: "source_shape", Type: model.TagTypeEnum,
			EnumValues: []string{"annular", "dipole_x", "dipole_y", "quasar", "freeform"}},
		{Name: "wavelength", Type: model.TagTypeFloat, Required: true},
		{Name: "na", Type: model.TagTypeFloat, Required: true},
		{Name: "sigma_in", Type: model.TagTypeFloat},
		{Name: "sigma_out", Type: model.TagTypeFloat},
		{Name: "polarization", Type: model.TagTypeEnum,
			EnumValues: []string{"unpolarized", "te", "tm", "xy"}},
		{Name: "flare", Type: model.TagTypeFloat},
		{Name: "mask3d", Type: model.TagTypeBool},
		{Name: "accuracy_rms", Type: model.TagTypeFloat},
		{Name: "anchor_points", Type: model.TagTypeInt},
		{Name: "status", Type: model.TagTypeEnum, Required: true,
			EnumValues: []string{"in_review", "released", "deprecated"}},
		{Name: "version", Type: model.TagTypeString},
		{Name: "owner", Type: model.TagTypeString},
		{Name: "calibration", Type: model.TagTypeObject, Fields: []model.TagDefinition{
			{Name: "train_points", Type: model.TagTypeInt, Required: true},
			{Name: "test_points", Type: model.TagTypeInt, Required: true},
			{Name: "residual_nm", Type: model.TagTypeFloat},
		}},
		{Name: "keywords", Type: model.TagTypeArray, ElementType: model.TagTypeString},
	}
}

func caseSchema() []model.TagDefinition {
	return []model.TagDefinition{
		{Name: "name", Type: model.TagTypeString, Required: true},
		{Name: "node", Type: model.TagTypeEnum, Required: true, EnumValues: nodes},
		{Name: "corner", Type: model.TagTypeEnum, Required: true,
			EnumValues: []string{"tt", "ff", "ss", "sf", "fs"}},
		{Name: "purpose", Type: model.TagTypeEnum, Required: true,
			EnumValues: []string{"verification", "characterization", "robustness", "drc_fix"}},
		{Name: "priority", Type: model.TagTypeEnum,
			EnumValues: []string{"p0", "p1", "p2", "p3"}},
		{Name: "status", Type: model.TagTypeEnum, Required: true,
			EnumValues: []string{"new", "running", "passed", "failed", "blocked"}},
		{Name: "expect_cd", Type: model.TagTypeFloat},
		{Name: "measured_cd", Type: model.TagTypeFloat},
		{Name: "cd_error", Type: model.TagTypeFloat},
		{Name: "meef", Type: model.TagTypeFloat},
		{Name: "dose", Type: model.TagTypeFloat},
		{Name: "focus", Type: model.TagTypeFloat},
		{Name: "wafer_count", Type: model.TagTypeInt},
		{Name: "start_date", Type: model.TagTypeDate},
		{Name: "owner", Type: model.TagTypeString},
		{Name: "keywords", Type: model.TagTypeArray, ElementType: model.TagTypeString},
	}
}

func layoutSchema() []model.TagDefinition {
	return []model.TagDefinition{
		{Name: "name", Type: model.TagTypeString, Required: true},
		{Name: "node", Type: model.TagTypeEnum, Required: true, EnumValues: nodes},
		{Name: "cell", Type: model.TagTypeString},
		{Name: "format", Type: model.TagTypeEnum, Required: true,
			EnumValues: []string{"gds", "oasis", "def"}},
		{Name: "file_size", Type: model.TagTypeInt},
		{Name: "area_um2", Type: model.TagTypeFloat},
		{Name: "density", Type: model.TagTypeFloat},
		{Name: "layer_count", Type: model.TagTypeInt},
		{Name: "drc_status", Type: model.TagTypeEnum,
			EnumValues: []string{"clean", "waive", "violate"}},
		{Name: "library", Type: model.TagTypeString},
		{Name: "modified", Type: model.TagTypeDate},
		{Name: "keywords", Type: model.TagTypeArray, ElementType: model.TagTypeString},
	}
}

func layerSchema() []model.TagDefinition {
	return []model.TagDefinition{
		{Name: "name", Type: model.TagTypeString, Required: true},
		{Name: "layer_type", Type: model.TagTypeEnum, Required: true,
			EnumValues: []string{"metal", "via", "poly", "active", "implant", "well", "cut", "seal"}},
		{Name: "purpose", Type: model.TagTypeEnum,
			EnumValues: []string{"drawing", "derived", "assist"}},
		{Name: "min_width", Type: model.TagTypeFloat},
		{Name: "min_space", Type: model.TagTypeFloat},
		{Name: "pitch", Type: model.TagTypeFloat},
		{Name: "density_min", Type: model.TagTypeFloat},
		{Name: "density_max", Type: model.TagTypeFloat},
		{Name: "opc_treatment", Type: model.TagTypeEnum, Required: true,
			EnumValues: []string{"none", "sb", "sraf", "ilt"}},
		{Name: "epe_violations", Type: model.TagTypeInt},
		{Name: "lvs_status", Type: model.TagTypeEnum,
			EnumValues: []string{"pass", "fail", "pending"}},
		{Name: "keywords", Type: model.TagTypeArray, ElementType: model.TagTypeString},
	}
}

// ---------- 数据生成 ----------

// buildItem 生成数据项并校验标签（复用服务层校验，保证与系统规则一致）
func buildItem(r *rand.Rand, colID, adminID primitive.ObjectID, schema []model.TagDefinition,
	path string, tags map[string]interface{}) (*model.DataItem, error) {

	normalized, err := service.ValidateAndNormalizeTags(schema, tags, false)
	if err != nil {
		return nil, fmt.Errorf("标签校验失败 %s: %v", path, err.Message)
	}
	it := &model.DataItem{
		ID:           primitive.NewObjectID(),
		CollectionID: colID,
		Path:         path,
		Tags:         normalized,
		ManualTags:   normalized,
		TagSource:    model.TagSourceManual,
		ScrapeStatus: model.ItemScrapeNone,
		CreatedBy:    adminID,
	}
	return it, nil
}

// genModel 生成一个 OPC 模型数据项
func genModel(r *rand.Rand, i int, colID, adminID primitive.ObjectID, schema []model.TagDefinition,
	root string) (*model.DataItem, error) {

	name := fmt.Sprintf("MOD_%04d", i)
	node := pickWeighted(r, nodes, nodeWeights)
	wavelength := 193.0
	if (node == "3" || node == "5" || node == "7") && r.Float64() < 0.7 {
		wavelength = 13.5 // EUV
	} else if r.Float64() < 0.1 {
		wavelength = 248.0
	}
	na := naByNode[node]
	if wavelength == 13.5 {
		na = 0.33
	} else {
		na = round2(na + (r.Float64()-0.5)*0.04)
	}
	sigmaIn := round2(0.5 + r.Float64()*0.3)
	sigmaOut := round2(math.Min(1.0, sigmaIn+0.3+r.Float64()*0.4))

	tags := map[string]interface{}{
		"name":           name,
		"node":           node,
		"model_type":     pickWeighted(r, []string{"optical", "resist", "etch", "full"}, []int{5, 3, 1, 1}),
		"lib_type":       pickWeighted(r, []string{"im", "abbe", "socs", "r3d"}, []int{3, 2, 4, 1}),
		"source_shape":   pickWeighted(r, []string{"annular", "dipole_x", "dipole_y", "quasar", "freeform"}, []int{4, 2, 2, 2, 1}),
		"wavelength":     wavelength,
		"na":             na,
		"sigma_in":       sigmaIn,
		"sigma_out":      sigmaOut,
		"polarization":   pickWeighted(r, []string{"unpolarized", "te", "tm", "xy"}, []int{3, 3, 1, 4}),
		"flare":          round2(r.Float64()*2.0),
		"mask3d":         r.Float64() < 0.6,
		"accuracy_rms":   round2(0.1 + r.Float64()*1.4),
		"anchor_points":  200 + r.Intn(1800),
		"status":         pickWeighted(r, []string{"in_review", "released", "deprecated"}, []int{3, 6, 1}),
		"version":        fmt.Sprintf("v%d.%d.%d", 1+r.Intn(2), r.Intn(9), r.Intn(10)),
		"owner":          pick(r, owners),
		"calibration": map[string]interface{}{
			"train_points": 300 + r.Intn(2700),
			"test_points":  100 + r.Intn(900),
			"residual_nm":  round2(0.05 + r.Float64()*0.45),
		},
		"keywords": pickKw(r, []string{"193i", "euv", "ilt", "smo", "sraf", "meef", "hotspot", "process-window"}),
	}

	path := filepath.Join(root, "litho", "model", name+".gds")
	return buildItem(r, colID, adminID, schema, path, tags)
}

// genLayout 生成一个版图数据项，并返回其下图层数（图层数据单独生成）
func genLayout(r *rand.Rand, i int, colID, adminID primitive.ObjectID, schema []model.TagDefinition,
	root string, layerN int) (*model.DataItem, error) {

	name := fmt.Sprintf("LAY_%04d", i)
	node := pickWeighted(r, nodes, nodeWeights)
	tags := map[string]interface{}{
		"name":        name,
		"node":        node,
		"cell":        "TOP_" + name,
		"format":      pickWeighted(r, []string{"gds", "oasis", "def"}, []int{8, 2, 1}),
		"file_size":   50_000_000 + r.Intn(1_950_000_000),
		"area_um2":    round2(5e6 + r.Float64()*195e6),
		"density":     round2(0.2 + r.Float64()*0.7),
		"layer_count": layerN,
		"drc_status":  pickWeighted(r, []string{"clean", "waive", "violate"}, []int{7, 2, 1}),
		"library":     pick(r, []string{"stdcell", "io", "sram", "analog", "custom"}),
		"modified":    randDate(r, 180),
		"keywords":    pickKw(r, []string{"full-chip", "block", "drc", "opc", "mp", "euv"}),
	}
	path := filepath.Join(root, "litho", "layout", name+".gds")
	return buildItem(r, colID, adminID, schema, path, tags)
}

// genLayer 生成一个图层数据项（版图的子项，物化路径写入父 id）
func genLayer(r *rand.Rand, layout *model.DataItem, j int, layerName string,
	colID, adminID primitive.ObjectID, schema []model.TagDefinition,
	root string) (*model.DataItem, error) {

	node := layout.Tags["node"].(string)
	pitch := pitchByNode[node] * (0.9 + r.Float64()*0.2)
	minWidth := pitch * (0.35 + r.Float64()*0.15)
	minSpace := pitch * (0.45 + r.Float64()*0.1)
	dMin := round2(0.1 + r.Float64()*0.2)
	dMax := round2(dMin + 0.3 + r.Float64()*0.4)

	tags := map[string]interface{}{
		"name":           layerName,
		"layer_type":     layerTypeOf(layerName),
		"purpose":        pickWeighted(r, []string{"drawing", "derived", "assist"}, []int{7, 2, 1}),
		"min_width":      round2(minWidth),
		"min_space":      round2(minSpace),
		"pitch":          round2(pitch),
		"density_min":    dMin,
		"density_max":    dMax,
		"opc_treatment":  pickWeighted(r, []string{"none", "sb", "sraf", "ilt"}, []int{2, 4, 3, 2}),
		"epe_violations": 0,
		"lvs_status":     pickWeighted(r, []string{"pass", "fail", "pending"}, []int{8, 1, 2}),
		"keywords":       pickKw(r, []string{"metal", "via", "opc", "epe", "drc", "pitch"}),
	}
	if r.Float64() < 0.3 {
		tags["epe_violations"] = 1 + r.Intn(14)
	}

	it, err := buildItem(r, colID, adminID, schema,
		filepath.Join(root, "litho", "layout", layout.Tags["name"].(string), layerName+".dxf"), tags)
	if err != nil {
		return nil, err
	}
	// 物化路径：父 = 所属版图（P3 优化，插入时直接写入）
	it.Ancestors = []primitive.ObjectID{layout.ID}
	return it, nil
}

// genCase 生成一个测试用例数据项
func genCase(r *rand.Rand, i int, colID, adminID primitive.ObjectID, schema []model.TagDefinition,
	root string) (*model.DataItem, error) {

	name := fmt.Sprintf("CASE_%04d", i)
	expect := round2(10 + r.Float64()*30)
	measured := round2(expect + (r.Float64()-0.5)*2)
	tags := map[string]interface{}{
		"name":         name,
		"node":         pickWeighted(r, nodes, nodeWeights),
		"corner":       pickWeighted(r, []string{"tt", "ff", "ss", "sf", "fs"}, []int{3, 2, 2, 2, 1}),
		"purpose":      pickWeighted(r, []string{"verification", "characterization", "robustness", "drc_fix"}, []int{4, 3, 2, 1}),
		"priority":     pickWeighted(r, []string{"p0", "p1", "p2", "p3"}, []int{2, 3, 3, 2}),
		"status":       pickWeighted(r, []string{"new", "running", "passed", "failed", "blocked"}, []int{2, 2, 4, 2, 1}),
		"expect_cd":    expect,
		"measured_cd":  measured,
		"cd_error":     round2(measured - expect),
		"meef":         round2(1 + r.Float64()*5),
		"dose":         round2(30 + r.Float64()*30),
		"focus":        round2((r.Float64() - 0.5) * 100),
		"wafer_count":  5 + r.Intn(45),
		"start_date":   randDate(r, 120),
		"owner":        pick(r, owners),
		"keywords":     pickKw(r, []string{"opc", "ilt", "pw", "meef", "cd", "hotspot", "verify"}),
	}
	path := filepath.Join(root, "litho", "case", name+".json")
	return buildItem(r, colID, adminID, schema, path, tags)
}

// ---------- 文件树 ----------

// createFiles 并发创建文件树（真实 NFS 模拟）
func createFiles(jobs []string, workers int, noFiles bool) error {
	if noFiles {
		return nil
	}
	total := int64(len(jobs))
	var done atomic.Int64
	ch := make(chan string)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for p := range ch {
				if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
					mu.Lock()
					if firstErr == nil {
						firstErr = fmt.Errorf("创建目录失败 %s: %w", p, err)
					}
					mu.Unlock()
					return
				}
				content := "placeholder generated by datacenter seeder\npath=" + p + "\n"
				if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
					mu.Lock()
					if firstErr == nil {
						firstErr = fmt.Errorf("写文件失败 %s: %w", p, err)
					}
					mu.Unlock()
					return
				}
				n := done.Add(1)
				if n%2000 == 0 {
					fmt.Printf("  [files] %d/%d\n", n, total)
				}
			}
		}()
	}
	for _, p := range jobs {
		ch <- p
	}
	close(ch)
	wg.Wait()
	return firstErr
}

// ---------- 主流程 ----------

func main() {
	var (
		cfgPath  = flag.String("config", "config/config.yaml", "配置文件路径")
		fresh    = flag.Bool("fresh", true, "先清空同名集合再重建（默认 true）")
		nModels  = flag.Int("models", 3000, "模型数量")
		nLayouts = flag.Int("layouts", 2000, "版图数量")
		nLayers  = flag.Int("layers", 0, "图层总数（0 = 每个版图随机 4~14 层）")
		nCases   = flag.Int("cases", 3000, "测试用例数量")
		workers  = flag.Int("workers", 16, "文件创建并发数")
		noFiles  = flag.Bool("no-files", false, "跳过真实文件创建")
		seed     = flag.Int64("seed", 0, "随机种子（0 = 按当前时间）")
	)
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载配置失败: %v\n", err)
		os.Exit(1)
	}
	if err := logger.Init(cfg.Log.Level, cfg.Log.Output); err != nil {
		fmt.Fprintf(os.Stderr, "初始化日志失败: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	ctx := context.Background()

	db, err := database.Connect(cfg.Database)
	if err != nil {
		fmt.Fprintf(os.Stderr, "MongoDB 连接失败: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		_ = db.Close(cctx)
	}()

	if err := database.EnsureIndexes(db.DB); err != nil {
		fmt.Fprintf(os.Stderr, "初始化索引失败: %v\n", err)
		os.Exit(1)
	}
	if err := database.EnsureBootstrapAdmin(db.DB, cfg.Bootstrap); err != nil {
		fmt.Fprintf(os.Stderr, "初始化管理员失败: %v\n", err)
		os.Exit(1)
	}
	users := store.NewUserStore(db.DB)
	admin, err := users.FindByUsername(ctx, cfg.Bootstrap.AdminUsername)
	if err != nil || admin == nil {
		fmt.Fprintf(os.Stderr, "找不到管理员 %s: %v\n", cfg.Bootstrap.AdminUsername, err)
		os.Exit(1)
	}
	adminID := admin.ID

	rndSeed := *seed
	if rndSeed == 0 {
		rndSeed = time.Now().UnixNano()
	}
	r := rand.New(rand.NewSource(rndSeed))
	fmt.Printf("== datacenter seeder ==  db=%s  seed=%d\n", cfg.Database.Name, rndSeed)
	fmt.Printf("models=%d layouts=%d layers=%d cases=%d fresh=%v workers=%d no-files=%v\n",
		*nModels, *nLayouts, *nLayers, *nCases, *fresh, *workers, *noFiles)

	cols := store.NewCollectionStore(db.DB)
	items := store.NewItemStore(db.DB)
	tasks := store.NewTaskStore(db.DB)
	rels := store.NewRelationStore(db.DB)

	// 1. 集合
	colDefs := []struct {
		name, desc string
		schema     []model.TagDefinition
	}{
		{"model", "OPC 光学/光刻胶/刻蚀模型（计算光刻 OPC 建模产出）", modelSchema()},
		{"case", "OPC 测试用例（验证/表征/鲁棒性/DVC 用例）", caseSchema()},
		{"layout", "版图设计数据（GDS/OASIS/DEF，全芯片或 block）", layoutSchema()},
		{"layer", "版图图层数据（金属/通孔/有源/多晶等，属版图的子项）", layerSchema()},
	}
	colMap := map[string]*model.BusinessCollection{}
	for _, def := range colDefs {
		c, err := ensureCollection(ctx, cols, items, tasks, rels, adminID, def.name, def.desc, def.schema, *fresh)
		if err != nil {
			fmt.Fprintf(os.Stderr, "准备集合 %s 失败: %v\n", def.name, err)
			os.Exit(1)
		}
		colMap[def.name] = c
		fmt.Printf("[collection] %-6s id=%s schema=%d\n", c.Name, c.ID.Hex(), len(c.TagSchema))
	}

	// 2. 数据项
	var (
		modelItems  []*model.DataItem
		layoutItems []*model.DataItem
		layerItems  []*model.DataItem
		caseItems   []*model.DataItem
		fileJobs    []string
	)
	layerN := layerCounts(r, *nLayouts, *nLayers)

	start := time.Now()
	fmt.Println("== 生成数据项 ==")

	for i := 0; i < *nModels; i++ {
		it, err := genModel(r, i, colMap["model"].ID, adminID, colMap["model"].TagSchema, cfg.Data.RootDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "生成模型失败: %v\n", err)
			os.Exit(1)
		}
		modelItems = append(modelItems, it)
		fileJobs = append(fileJobs, it.Path)
	}
	fmt.Printf("[gen] model: %d\n", len(modelItems))

	for i := 0; i < *nLayouts; i++ {
		it, err := genLayout(r, i, colMap["layout"].ID, adminID, colMap["layout"].TagSchema, cfg.Data.RootDir, layerN[i])
		if err != nil {
			fmt.Fprintf(os.Stderr, "生成版图失败: %v\n", err)
			os.Exit(1)
		}
		layoutItems = append(layoutItems, it)
		fileJobs = append(fileJobs, it.Path)
	}
	fmt.Printf("[gen] layout: %d\n", len(layoutItems))

	for i, lay := range layoutItems {
		names := pickN(r, layerPool, layerN[i])
		for j, nm := range names {
			it, err := genLayer(r, lay, j, nm, colMap["layer"].ID, adminID, colMap["layer"].TagSchema, cfg.Data.RootDir)
			if err != nil {
				fmt.Fprintf(os.Stderr, "生成图层失败: %v\n", err)
				os.Exit(1)
			}
			layerItems = append(layerItems, it)
			fileJobs = append(fileJobs, it.Path)
		}
	}
	fmt.Printf("[gen] layer: %d\n", len(layerItems))

	for i := 0; i < *nCases; i++ {
		it, err := genCase(r, i, colMap["case"].ID, adminID, colMap["case"].TagSchema, cfg.Data.RootDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "生成用例失败: %v\n", err)
			os.Exit(1)
		}
		caseItems = append(caseItems, it)
		fileJobs = append(fileJobs, it.Path)
	}
	fmt.Printf("[gen] case: %d\n", len(caseItems))
	fmt.Printf("[gen] 数据项合计: %d，文件: %d，耗时 %v\n",
		len(modelItems)+len(layoutItems)+len(layerItems)+len(caseItems), len(fileJobs), time.Since(start).Round(time.Millisecond))

	// 3. 文件树
	start = time.Now()
	fmt.Println("== 创建文件树 ==")
	if err := createFiles(fileJobs, *workers, *noFiles); err != nil {
		fmt.Fprintf(os.Stderr, "创建文件树失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("[files] 完成 %d 个文件，耗时 %v\n", len(fileJobs), time.Since(start).Round(time.Millisecond))

	// 4. 批量入库
	start = time.Now()
	fmt.Println("== 写入 MongoDB ==")
	insertItems := func(coll string, list []*model.DataItem) {
		for i := 0; i < len(list); i += 500 {
			end := i + 500
			if end > len(list) {
				end = len(list)
			}
			if err := items.InsertMany(ctx, list[i:end]); err != nil {
				fmt.Fprintf(os.Stderr, "写入 %s 失败（%d~%d）: %v\n", coll, i, end, err)
				os.Exit(1)
			}
		}
		fmt.Printf("[insert] %-6s %d\n", coll, len(list))
	}
	insertItems("model", modelItems)
	insertItems("layout", layoutItems)
	insertItems("layer", layerItems)
	insertItems("case", caseItems)
	fmt.Printf("[insert] 耗时 %v\n", time.Since(start).Round(time.Millisecond))

	// 5. 关联关系：版图→图层（父子），用例→模型/版图（引用）
	start = time.Now()
	fmt.Println("== 写入关联关系 ==")

	var edges []*model.Relation
	// 父子：from=版图(父) → to=图层(子)；图层按父版图分组（Ancestors[0] 即父 id）
	childrenOf := map[string][]*model.DataItem{}
	for _, it := range layerItems {
		pid := it.Ancestors[0].Hex()
		childrenOf[pid] = append(childrenOf[pid], it)
	}
	for _, lay := range layoutItems {
		for j, child := range childrenOf[lay.ID.Hex()] {
			edges = append(edges, &model.Relation{
				CollectionID: colMap["layout"].ID,
				FromItemID:   lay.ID,
				ToItemID:     child.ID,
				Type:         model.RelationParentChild,
				Meta:         map[string]interface{}{"layer_order": j + 1},
				CreatedBy:    adminID,
			})
		}
	}
	// 引用：用例 → 模型 / 用例 → 版图
	for i, c := range caseItems {
		m := modelItems[i%len(modelItems)]
		edges = append(edges, &model.Relation{
			CollectionID: colMap["case"].ID,
			FromItemID:   c.ID,
			ToItemID:     m.ID,
			Type:         model.RelationReference,
			Meta:         map[string]interface{}{"usage": "opc_model"},
			CreatedBy:    adminID,
		})
		l := layoutItems[i%len(layoutItems)]
		edges = append(edges, &model.Relation{
			CollectionID: colMap["case"].ID,
			FromItemID:   c.ID,
			ToItemID:     l.ID,
			Type:         model.RelationReference,
			Meta:         map[string]interface{}{"usage": "layout"},
			CreatedBy:    adminID,
		})
	}
	for i := 0; i < len(edges); i += 1000 {
		end := i + 1000
		if end > len(edges) {
			end = len(edges)
		}
		if err := rels.InsertMany(ctx, edges[i:end]); err != nil {
			fmt.Fprintf(os.Stderr, "写入关系失败（%d~%d）: %v\n", i, end, err)
			os.Exit(1)
		}
	}
	fmt.Printf("[insert] 关系 %d 条（父子 %d，引用 %d），耗时 %v\n", len(edges),
		len(layerItems), len(caseItems)*2, time.Since(start).Round(time.Millisecond))

	// 6. 汇总校验
	fmt.Println("== 汇总 ==")
	totalItems := int64(0)
	for _, c := range colDefs {
		n, err := items.Count(ctx, bson.M{"collection_id": colMap[c.name].ID})
		if err != nil {
			fmt.Fprintf(os.Stderr, "统计失败: %v\n", err)
			os.Exit(1)
		}
		totalItems += n
		fmt.Printf("  %-6s items=%d\n", c.name, n)
	}
	pc, _ := rels.Count(ctx, bson.M{"type": model.RelationParentChild})
	rc, _ := rels.Count(ctx, bson.M{"type": model.RelationReference})
	cc, _ := rels.Count(ctx, bson.M{"type": model.RelationCall})
	anc, _ := items.Count(ctx, bson.M{"ancestors.0": bson.M{"$exists": true}})
	fmt.Printf("  数据项合计=%d 关系合计=%d（parent_child=%d reference=%d call=%d）物化路径(ancestors)=%d\n",
		totalItems, pc+rc+cc, pc, rc, cc, anc)
	fmt.Printf("== 完成，总耗时 %v ==\n", time.Since(start).Round(time.Millisecond))
}

// ensureCollection 准备集合：-fresh 时清空并重建；否则复用已存在的同名集合
func ensureCollection(ctx context.Context, cols *store.CollectionStore, items *store.ItemStore,
	tasks *store.TaskStore, rels *store.RelationStore, adminID primitive.ObjectID,
	name, desc string, schema []model.TagDefinition, fresh bool) (*model.BusinessCollection, error) {

	if err := service.ValidateTagSchema(schema); err != nil {
		return nil, err
	}
	if fresh {
		existing, _, err := cols.List(ctx, bson.M{"name": name}, 1, 10)
		if err != nil {
			return nil, err
		}
		for _, c := range existing {
			if err := items.DeleteByCollection(ctx, c.ID); err != nil {
				return nil, err
			}
			if err := rels.DeleteByCollection(ctx, c.ID); err != nil {
				return nil, err
			}
			if err := tasks.DeleteByCollection(ctx, c.ID); err != nil {
				return nil, err
			}
			if err := cols.Delete(ctx, c.ID); err != nil {
				return nil, err
			}
			fmt.Printf("[wipe] 清空集合 %s\n", name)
		}
	} else {
		existing, _, err := cols.List(ctx, bson.M{"name": name}, 1, 10)
		if err != nil {
			return nil, err
		}
		if len(existing) > 0 {
			return existing[0], nil
		}
	}
	c := &model.BusinessCollection{
		Name:        name,
		Description: desc,
		CreatedBy:   adminID,
		TagSchema:   schema,
		Members: []model.Member{
			{UserID: adminID, Role: model.MemberRoleCollectionAdmin},
		},
	}
	if err := cols.Create(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}
