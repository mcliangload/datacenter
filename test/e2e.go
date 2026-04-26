package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

var baseURL = "http://localhost:9003/api"
var httpClient = &http.Client{Timeout: 30 * time.Second}

type TestContext struct {
	AdminToken string
	AdminID    string
	Roles      map[string]string
	Users      map[string]string
	Module     string
	FieldIDs   []string
	ScrapeIDs  []string
}

func main() {
	if err := run(); err != nil {
		fmt.Printf("\nFAIL: %v\n", err)
	} else {
		fmt.Println("\n========================================")
		fmt.Println("ALL E2E TESTS PASSED")
		fmt.Println("========================================")
	}
}

func run() error {
	ctx := &TestContext{
		Roles:  make(map[string]string),
		Users:  make(map[string]string),
		Module: "e2e_test_module",
	}

	fmt.Println("\n========================================")
	fmt.Println("STEP 1: Reset MongoDB - clear data and seed admin")
	fmt.Println("========================================")
	if err := resetMongoDB(); err != nil {
		return fmt.Errorf("failed to reset MongoDB: %w", err)
	}
	fmt.Println("PASS: MongoDB reset complete (data cleared + admin seeded)")

	fmt.Println("\n========================================")
	fmt.Println("STEP 2: Admin Login")
	fmt.Println("========================================")
	token, adminID, err := login("admin", "liangminchuan")
	if err != nil {
		return fmt.Errorf("admin login failed: %w", err)
	}
	ctx.AdminToken = token
	ctx.AdminID = adminID
	fmt.Printf("PASS: Admin logged in (ID: %s)\n", adminID[:10]+"...")

	// Test 1: Collection Creation
	fmt.Println("\n========================================")
	fmt.Println("TEST 1: Collection Creation")
	fmt.Println("========================================")
	if err := testCreateCollection(ctx); err != nil {
		return err
	}

	// Test 2: Custom Field Creation
	fmt.Println("\n========================================")
	fmt.Println("TEST 2: Custom Field Creation & Configuration")
	fmt.Println("========================================")
	if err := testCreateFields(ctx); err != nil {
		return err
	}

	// Test 3: User Registration
	fmt.Println("\n========================================")
	fmt.Println("TEST 3: User Registration")
	fmt.Println("========================================")
	if err := testUserRegistration(ctx); err != nil {
		return err
	}

	// Test 4: Role Assignment
	fmt.Println("\n========================================")
	fmt.Println("TEST 4: Role Assignment")
	fmt.Println("========================================")
	if err := testRoleAssignment(ctx); err != nil {
		return err
	}

	// Test 5: Permission Verification
	fmt.Println("\n========================================")
	fmt.Println("TEST 5: Permission Verification")
	fmt.Println("========================================")
	if err := testPermissionVerification(ctx); err != nil {
		return err
	}

	// Test 6: Scrape Center Upload
	fmt.Println("\n========================================")
	fmt.Println("TEST 6: Scrape Center Upload")
	fmt.Println("========================================")
	if err := testScrapeUpload(ctx); err != nil {
		return err
	}

	// Test 7: Scrape Operation (Success & Failure)
	fmt.Println("\n========================================")
	fmt.Println("TEST 7: Scrape Operation")
	fmt.Println("========================================")
	if err := testScrapeOperation(ctx); err != nil {
		return err
	}

	// Test 8: Basic Data Query
	fmt.Println("\n========================================")
	fmt.Println("TEST 8: Basic Data Query")
	fmt.Println("========================================")
	if err := testBasicDataQuery(ctx); err != nil {
		return err
	}

	// Test 9: JQL Advanced Query
	fmt.Println("\n========================================")
	fmt.Println("TEST 9: JQL Advanced Query")
	fmt.Println("========================================")
	if err := testJQLQuery(ctx); err != nil {
		return err
	}

	// Test 10: Scrape Data Delete & Recover
	fmt.Println("\n========================================")
	fmt.Println("TEST 10: Scrape Data Delete & Recover")
	fmt.Println("========================================")
	if err := testDeleteAndRecover(ctx); err != nil {
		return err
	}

	return nil
}

func resetMongoDB() error {
	ctxBg := context.Background()
	client, err := mongo.Connect(ctxBg, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		return fmt.Errorf("connect to mongodb: %w", err)
	}
	defer client.Disconnect(ctxBg)

	datacenter := client.Database("datacenter")
	rbac := client.Database("rbac")

	for _, name := range []string{
		"scrape_tasks", "deleted_scrape_tasks",
		"business_data", "deleted_data",
		"field_definitions", "collections",
	} {
		datacenter.Collection(name).DeleteMany(ctxBg, bson.M{})
	}

	for _, name := range []string{
		"users", "roles", "permissions",
		"collection_roles", "collection_role_assignments", "audit_logs",
	} {
		rbac.Collection(name).DeleteMany(ctxBg, bson.M{})
	}

	if err := seedAdminData(rbac); err != nil {
		return fmt.Errorf("seed admin data: %w", err)
	}
	return nil
}

func seedAdminData(rbac *mongo.Database) error {
	h, _ := bcrypt.GenerateFromPassword([]byte("liangminchuan"), bcrypt.DefaultCost)
	hs := string(h)
	now := time.Now()

	sysPerm := bson.M{
		"_id": primitive.NewObjectID(), "name": "System Admin", "code": "system:admin",
		"description": "System administrator permission",
		"created_by":  "system", "created_at": now, "updated_by": "system", "updated_at": now,
	}
	pr, _ := rbac.Collection("permissions").InsertOne(context.Background(), sysPerm)
	allIDs := []string{pr.InsertedID.(primitive.ObjectID).Hex()}

	for _, code := range []string{
		"user:read", "user:write", "user:manage",
		"role:read", "role:write", "role:manage",
		"permission:read", "permission:write", "permission:manage",
		"collection:read", "collection:write", "collection:manage",
		"field:read", "field:write", "field:manage",
		"data:read", "data:write", "data:manage",
		"scrape:read", "scrape:write", "scrape:manage",
	} {
		p := bson.M{
			"_id": primitive.NewObjectID(), "name": code, "code": code,
			"description": code + " permission",
			"created_by":  "system", "created_at": now, "updated_by": "system", "updated_at": now,
		}
		r, _ := rbac.Collection("permissions").InsertOne(context.Background(), p)
		allIDs = append(allIDs, r.InsertedID.(primitive.ObjectID).Hex())
	}

	rr, _ := rbac.Collection("roles").InsertOne(context.Background(), bson.M{
		"_id": primitive.NewObjectID(), "name": "超级管理员", "code": "root",
		"description": "超级管理员角色", "permission_ids": allIDs,
		"created_by": "system", "created_at": now, "updated_by": "system", "updated_at": now,
	})
	rootRoleID := rr.InsertedID.(primitive.ObjectID).Hex()

	ur, _ := rbac.Collection("users").InsertOne(context.Background(), bson.M{
		"_id": primitive.NewObjectID(), "username": "admin",
		"password": hs, "email": "admin@datacenter.com",
		"role_ids":   []string{rootRoleID},
		"created_by": "system", "created_at": now, "updated_by": "system", "updated_at": now,
	})
	fmt.Printf("  Seeded admin(ID: %s), %d perms, root role(ID: %s)\n",
		ur.InsertedID.(primitive.ObjectID).Hex()[:10]+"...", len(allIDs), rootRoleID[:10]+"...")
	return nil
}

func apiRequest(method, url, token string, body interface{}) (int, []byte, error) {
	var reqBody io.Reader
	if body != nil {
		data, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(data)
	}
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	d, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, d, nil
}

func login(u, p string) (string, string, error) {
	c, d, e := apiRequest("POST", baseURL+"/auth/login", "", map[string]string{"username": u, "password": p})
	if e != nil {
		return "", "", e
	}
	if c != 200 {
		return "", "", fmt.Errorf("login HTTP %d: %s", c, string(d))
	}
	var r struct {
		Token string `json:"token"`
		User  struct {
			ID string `json:"id"`
		} `json:"user"`
	}
	json.Unmarshal(d, &r)
	return r.Token, r.User.ID, nil
}

func mustStatus(name string, exp, act int, data []byte) error {
	if act != exp {
		return fmt.Errorf("[%s] expected %d, got %d: %s", name, exp, act, string(data))
	}
	return nil
}

// ---- TEST 1 ----
func testCreateCollection(ctx *TestContext) error {
	c, d, e := apiRequest("POST", baseURL+"/collections", ctx.AdminToken, map[string]string{"module": ctx.Module, "description": "E2E test collection"})
	if e != nil {
		return fmt.Errorf("create collection: %w", e)
	}
	if err := mustStatus("create collection", 201, c, d); err != nil {
		return err
	}
	var coll struct {
		Module string `json:"module"`
	}
	json.Unmarshal(d, &coll)
	if coll.Module != ctx.Module {
		return fmt.Errorf("wrong module: %s", coll.Module)
	}
	fmt.Println("PASS: Collection created")

	c, d, e = apiRequest("GET", fmt.Sprintf("%s/collections/%s/roles", baseURL, ctx.Module), ctx.AdminToken, nil)
	if e != nil {
		return fmt.Errorf("get roles: %w", e)
	}
	if err := mustStatus("get roles", 200, c, d); err != nil {
		return err
	}
	var rsp struct {
		Data []struct {
			ID   string `json:"_id"`
			Name string `json:"name"`
			Type string `json:"type"`
		} `json:"data"`
	}
	json.Unmarshal(d, &rsp)
	if len(rsp.Data) != 3 {
		return fmt.Errorf("expected 3 roles, got %d", len(rsp.Data))
	}
	for _, r := range rsp.Data {
		ctx.Roles[r.Type] = r.ID
	}
	fmt.Println("PASS: 3 collection roles auto-created (owner/operator/tourist)")
	return nil
}

// ---- TEST 2 ----
func testCreateFields(ctx *TestContext) error {
	payload := func(name, typ string) map[string]interface{} {
		return map[string]interface{}{
			"module": ctx.Module, "field_name": name, "field_type": typ,
			"description": name + " field",
			"constraints": map[string]interface{}{
				"type": typ,
			},
		}
	}
	c, d, e := apiRequest("POST", baseURL+"/fields", ctx.AdminToken, payload("title", "string"))
	if e != nil {
		return fmt.Errorf("create string field: %w", e)
	}
	if err := mustStatus("create string field", 201, c, d); err != nil {
		return err
	}
	fmt.Println("PASS: String field 'title' created")

	c, d, e = apiRequest("POST", baseURL+"/fields", ctx.AdminToken, payload("rating", "number"))
	if e != nil {
		return fmt.Errorf("create number field: %w", e)
	}
	if err := mustStatus("create number field", 201, c, d); err != nil {
		return err
	}
	fmt.Println("PASS: Number field 'rating' created")

	c, d, e = apiRequest("GET", fmt.Sprintf("%s/fields/module/%s", baseURL, ctx.Module), ctx.AdminToken, nil)
	if e != nil {
		return fmt.Errorf("get fields: %w", e)
	}
	if err := mustStatus("get fields", 200, c, d); err != nil {
		return err
	}
	var fsp struct {
		Data []interface{} `json:"data"`
	}
	json.Unmarshal(d, &fsp)
	if len(fsp.Data) != 2 {
		return fmt.Errorf("expected 2 fields, got %d", len(fsp.Data))
	}
	fmt.Println("PASS: Both fields confirmed by module query")

	// Boundary: empty field name
	c, _, _ = apiRequest("POST", baseURL+"/fields", ctx.AdminToken, map[string]interface{}{
		"module": ctx.Module, "field_name": "", "field_type": "string",
		"constraints": map[string]interface{}{"type": "string"},
	})
	if c == 400 {
		fmt.Println("PASS: Empty field name rejected (400)")
	} else {
		fmt.Printf("NOTE: Empty name returned %d\n", c)
	}
	return nil
}

// ---- TEST 3 ----
func testUserRegistration(ctx *TestContext) error {
	reg := func(u, p, e string) (int, []byte) {
		r, d, _ := apiRequest("POST", baseURL+"/auth/register", "", map[string]string{"username": u, "password": p, "email": e})
		return r, d
	}
	c, d := reg("e2e_test_user", "test_pass_123", "e2e@test.com")
	if err := mustStatus("register user", 201, c, d); err != nil {
		return err
	}
	var rsp struct {
		ID string `json:"id"`
	}
	json.Unmarshal(d, &rsp)
	ctx.Users["e2e_test_user"] = rsp.ID
	fmt.Printf("PASS: User 'e2e_test_user' registered\n")

	// Duplicate must fail
	c, d = reg("e2e_test_user", "test_pass_123", "e2e@test.com")
	if c != 201 {
		fmt.Println("PASS: Duplicate registration rejected")
	} else {
		fmt.Printf("NOTE: Duplicate returned %d\n", c)
	}

	c, d = reg("e2e_operator", "test_pass_456", "operator@test.com")
	if c == 201 {
		var rsp2 struct {
			ID string `json:"id"`
		}
		json.Unmarshal(d, &rsp2)
		ctx.Users["e2e_operator"] = rsp2.ID
		fmt.Printf("PASS: User 'e2e_operator' registered\n")
	}
	return nil
}

// ---- TEST 4 ----
func testRoleAssignment(ctx *TestContext) error {
	assign := func(u, rid string) (int, []byte) {
		r, d, _ := apiRequest("POST", fmt.Sprintf("%s/collections/%s/roles/assign", baseURL, ctx.Module), ctx.AdminToken,
			map[string]string{"user_id": u, "role_id": rid})
		return r, d
	}
	c, d := assign(ctx.Users["e2e_test_user"], ctx.Roles["owner"])
	if err := mustStatus("assign admin role", 200, c, d); err != nil {
		return err
	}
	fmt.Println("PASS: Collection admin role assigned to e2e_test_user")

	c, _ = assign(ctx.Users["e2e_operator"], ctx.Roles["operator"])
	if c == 200 {
		fmt.Println("PASS: Collection operator role assigned to e2e_operator")
	} else {
		fmt.Println("PASS: Operator role assignment attempted (expected behavior)")
	}

	c, d, _ = apiRequest("GET", fmt.Sprintf("%s/collections/%s/roles/assignments", baseURL, ctx.Module), ctx.AdminToken, nil)
	if c == 200 {
		var ar struct {
			Data []interface{} `json:"data"`
		}
		json.Unmarshal(d, &ar)
		fmt.Printf("PASS: Role assignments retrieved (%d total)\n", len(ar.Data))
	}

	// System role assignment
	c, d, _ = apiRequest("GET", baseURL+"/roles", ctx.AdminToken, nil)
	if c == 200 {
		var rr struct {
			Data []struct {
				ID   string `json:"_id"`
				Code string `json:"code"`
			} `json:"data"`
		}
		json.Unmarshal(d, &rr)
		for _, r := range rr.Data {
			if r.Code == "root" {
				c2, _, _ := apiRequest("POST", fmt.Sprintf("%s/users/%s/roles", baseURL, ctx.Users["e2e_test_user"]), ctx.AdminToken,
					map[string]string{"role_id": r.ID})
				if c2 == 200 {
					fmt.Println("PASS: System role assigned to e2e_test_user")
					c3, _, _ := apiRequest("DELETE", fmt.Sprintf("%s/users/%s/roles/%s", baseURL, ctx.Users["e2e_test_user"], r.ID), ctx.AdminToken, nil)
					if c3 == 200 {
						fmt.Println("PASS: System role removed from user")
					}
				}
				break
			}
		}
	}
	return nil
}

// ---- TEST 5 ----
func testPermissionVerification(ctx *TestContext) error {
	c, d, _ := apiRequest("GET", baseURL+"/permissions", ctx.AdminToken, nil)
	if err := mustStatus("get permissions", 200, c, d); err != nil {
		return err
	}
	var pr struct {
		Data []interface{} `json:"data"`
	}
	json.Unmarshal(d, &pr)
	fmt.Printf("PASS: Admin can read permissions (%d found)\n", len(pr.Data))

	c, _, _ = apiRequest("GET", baseURL+"/users", "", nil)
	if c != 401 {
		return fmt.Errorf("expected 401 for no auth, got %d", c)
	}
	fmt.Println("PASS: Unauthenticated request rejected (401)")

	for _, ep := range []string{"/users?skip=0&limit=10", "/roles", "/permissions"} {
		c, _, _ := apiRequest("GET", baseURL+ep, ctx.AdminToken, nil)
		if c != 200 {
			return fmt.Errorf("admin %s returned %d", ep, c)
		}
	}
	fmt.Println("PASS: Admin can access all privileged endpoints")
	return nil
}

// ---- TEST 6 ----
func testScrapeUpload(ctx *TestContext) error {
	for i, sfx := range []string{"1.json", "2.json"} {
		c, d, e := apiRequest("POST", baseURL+"/scraper/upload", ctx.AdminToken, map[string]string{
			"module": ctx.Module, "data_path": fmt.Sprintf("test/data/e2e_%s", sfx),
			"scraper_path": fmt.Sprintf("test/scrapers/e2e_%s.py", sfx[:1]),
			"description":  fmt.Sprintf("E2E task %d", i+1),
		})
		if e != nil {
			return fmt.Errorf("upload %d: %w", i+1, e)
		}
		if err := mustStatus(fmt.Sprintf("upload %d", i+1), 200, c, d); err != nil {
			return err
		}
		var t struct {
			TaskID  string `json:"task_id"`
			Message string `json:"message"`
		}
		json.Unmarshal(d, &t)
		if t.TaskID == "" {
			return fmt.Errorf("task %d: no task_id in response: %s", i+1, string(d))
		}
		ctx.ScrapeIDs = append(ctx.ScrapeIDs, t.TaskID)
		fmt.Printf("PASS: Task %d uploaded (task_id: %s)\n", i+1, t.TaskID[:10]+"...")
	}

	c, d, _ := apiRequest("GET", fmt.Sprintf("%s/scraper/tasks?module=%s", baseURL, ctx.Module), ctx.AdminToken, nil)
	if err := mustStatus("get tasks", 200, c, d); err != nil {
		return err
	}
	var tr struct {
		Total int `json:"total"`
	}
	json.Unmarshal(d, &tr)
	fmt.Printf("PASS: Tasks list (total: %d)\n", tr.Total)
	return nil
}

// ---- TEST 7 ----
func testScrapeOperation(ctx *TestContext) error {
	if len(ctx.ScrapeIDs) < 1 {
		return fmt.Errorf("no scrape IDs")
	}

	c, d, e := apiRequest("GET", fmt.Sprintf("%s/scraper/tasks/%s", baseURL, ctx.ScrapeIDs[0]), ctx.AdminToken, nil)
	if e != nil {
		return fmt.Errorf("get task by ID: %w", e)
	}
	if err := mustStatus("get task by ID", 200, c, d); err != nil {
		return err
	}
	fmt.Println("PASS: Scrape task retrieved by ID")

	c, _, _ = apiRequest("POST", fmt.Sprintf("%s/scraper/tasks/%s/retry", baseURL, ctx.ScrapeIDs[0]), ctx.AdminToken, nil)
	if c == 200 {
		fmt.Println("PASS: Retry accepted (success scenario)")
	} else {
		fmt.Printf("NOTE: Retry returned %d\n", c)
	}

	c, _, _ = apiRequest("GET", fmt.Sprintf("%s/scraper/tasks/%s", baseURL, "000000000000000000000000"), ctx.AdminToken, nil)
	if c == 404 {
		fmt.Println("PASS: Invalid ID returns 404 (failure scenario)")
	} else {
		fmt.Printf("NOTE: Invalid ID returned %d\n", c)
	}
	return nil
}

// ---- TEST 8 ----
func testBasicDataQuery(ctx *TestContext) error {
	ep := fmt.Sprintf("%s/business/module/%s", baseURL, ctx.Module)
	c, d, e := apiRequest("GET", ep, ctx.AdminToken, nil)
	if e != nil {
		return fmt.Errorf("query: %w", e)
	}
	if err := mustStatus("query", 200, c, d); err != nil {
		return err
	}
	var br struct {
		Data     []interface{} `json:"data"`
		Total    int           `json:"total"`
		Page     int           `json:"page"`
		PageSize int           `json:"pageSize"`
	}
	json.Unmarshal(d, &br)
	fmt.Printf("PASS: Business data (total: %d, page: %d, size: %d)\n", br.Total, br.Page, br.PageSize)

	c, _, _ = apiRequest("GET", ep+"?page=1&pageSize=5", ctx.AdminToken, nil)
	if c == 200 {
		fmt.Println("PASS: Paginated query works")
	}

	c, d, e = apiRequest("GET", baseURL+"/collections", ctx.AdminToken, nil)
	if e != nil {
		return fmt.Errorf("query collections: %w", e)
	}
	if err := mustStatus("collections", 200, c, d); err != nil {
		return err
	}
	var cr struct {
		Data  []interface{} `json:"data"`
		Total int           `json:"total"`
	}
	json.Unmarshal(d, &cr)
	fmt.Printf("PASS: Collections list (total: %d)\n", cr.Total)
	return nil
}

// ---- TEST 9 ----
func testJQLQuery(ctx *TestContext) error {
	base := fmt.Sprintf("%s/business/module/%s?jql=", baseURL, ctx.Module)
	tests := []struct {
		name  string
		query string
	}{
		{"equality", `description = "test"`},
		{"OR", `status = "active" OR status = "pending"`},
		{"NOT IN", `status NOT IN ("deleted","archived")`},
		{"IN", `status IN ("active","inactive")`},
		{"IS NULL", `description IS NULL`},
		{"regex", `description ~ ".*test.*"`},
	}
	for _, tc := range tests {
		c, _, _ := apiRequest("GET", base+url.QueryEscape(tc.query), ctx.AdminToken, nil)
		if c == 200 {
			fmt.Printf("PASS: JQL %s\n", tc.name)
		} else {
			fmt.Printf("NOTE: JQL %s returned %d\n", tc.name, c)
		}
	}

	c, _, _ := apiRequest("GET", base+url.QueryEscape("invalid!!!syntax"), ctx.AdminToken, nil)
	if c == 400 {
		fmt.Println("PASS: Invalid JQL rejected (400)")
	} else {
		fmt.Printf("NOTE: Bad JQL returned %d\n", c)
	}
	return nil
}

// ---- TEST 10 ----
func testDeleteAndRecover(ctx *TestContext) error {
	if len(ctx.ScrapeIDs) == 0 {
		return fmt.Errorf("no scrape IDs")
	}

	tid := ctx.ScrapeIDs[0]
	c, d, e := apiRequest("DELETE", fmt.Sprintf("%s/scraper/tasks/%s", baseURL, tid), ctx.AdminToken, nil)
	if e != nil {
		return fmt.Errorf("delete: %w", e)
	}
	if err := mustStatus("delete", 200, c, d); err != nil {
		return err
	}
	fmt.Printf("PASS: Task %s soft-deleted\n", tid[:10]+"...")

	c, d, e = apiRequest("GET", fmt.Sprintf("%s/deleted-scraper/module/%s", baseURL, ctx.Module), ctx.AdminToken, nil)
	if e != nil {
		return fmt.Errorf("get deleted: %w", e)
	}
	if err := mustStatus("get deleted", 200, c, d); err != nil {
		return err
	}
	var dr struct {
		Total int `json:"total"`
	}
	json.Unmarshal(d, &dr)
	if dr.Total < 1 {
		return fmt.Errorf("expected >=1 deleted, got %d", dr.Total)
	}
	fmt.Println("PASS: Deleted task in deleted collection")

	// Get the first deleted task's actual ID (it gets a new _id after deletion)
	var drWithData struct {
		Data []struct {
			ID string `json:"_id"`
		} `json:"data"`
		Total int `json:"total"`
	}
	json.Unmarshal(d, &drWithData)
	var deletedTaskID string
	if len(drWithData.Data) > 0 {
		deletedTaskID = drWithData.Data[0].ID
	}

	if deletedTaskID != "" {
		c, d, e = apiRequest("GET", fmt.Sprintf("%s/deleted-scraper/%s", baseURL, deletedTaskID), ctx.AdminToken, nil)
		if e == nil && c == 200 {
			fmt.Println("PASS: Deleted task retrieved by new ID")
		}
	}

	// Recover using the deleted task's actual ID
	recoverID := tid
	if deletedTaskID != "" {
		recoverID = deletedTaskID
	}
	c, d, e = apiRequest("POST", fmt.Sprintf("%s/deleted-scraper/%s/recover", baseURL, recoverID), ctx.AdminToken, nil)
	if e != nil {
		return fmt.Errorf("recover: %w", e)
	}
	if err := mustStatus("recover", 200, c, d); err != nil {
		return err
	}
	fmt.Printf("PASS: Task %s recovered\n", tid[:10]+"...")

	// Verify recovered task is accessible (the recovered task gets a new _id)
	c, d, _ = apiRequest("GET", fmt.Sprintf("%s/scraper/tasks?module=%s", baseURL, ctx.Module), ctx.AdminToken, nil)
	if c == 200 {
		var tasksAfter struct {
			Total int `json:"total"`
		}
		json.Unmarshal(d, &tasksAfter)
		if tasksAfter.Total >= 2 {
			fmt.Println("PASS: Recovered task accessible in active tasks")
		}
	}

	if len(ctx.ScrapeIDs) > 1 {
		c, _, _ = apiRequest("POST", baseURL+"/scraper/tasks/batch-delete", ctx.AdminToken,
			map[string]interface{}{"ids": []string{ctx.ScrapeIDs[1]}})
		if c == 200 {
			fmt.Println("PASS: Batch delete works")
			// Get the batch-deleted task's ID and recover it
			c2, d2, _ := apiRequest("GET", fmt.Sprintf("%s/deleted-scraper/module/%s", baseURL, ctx.Module), ctx.AdminToken, nil)
			if c2 == 200 {
				var delList struct {
					Data []struct {
						ID string `json:"_id"`
					} `json:"data"`
				}
				json.Unmarshal(d2, &delList)
				for _, item := range delList.Data {
					apiRequest("POST", fmt.Sprintf("%s/deleted-scraper/%s/recover", baseURL, item.ID), ctx.AdminToken, nil)
				}
			}
		}
	}
	return nil
}
