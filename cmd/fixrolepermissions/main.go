package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(context.Background())

	if err := client.Ping(ctx, nil); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Connected to MongoDB!")

	db := client.Database("rbac")

	permissionsCollection := db.Collection("permissions")
	rolesCollection := db.Collection("roles")

	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 获取所有权限
	var permissions []struct {
		ID   primitive.ObjectID `bson:"_id"`
		Code string             `bson:"code"`
		Name string             `bson:"name"`
	}

	cursor, err := permissionsCollection.Find(ctx, bson.M{})
	if err != nil {
		log.Fatal(err)
	}
	defer cursor.Close(ctx)

	if err := cursor.All(ctx, &permissions); err != nil {
		log.Fatal(err)
	}

	if len(permissions) == 0 {
		fmt.Println("No permissions found in the database!")
		return
	}

	fmt.Printf("Found %d permissions\n", len(permissions))
	for _, p := range permissions {
		fmt.Printf("  - %s (%s): %s\n", p.ID.Hex(), p.Code, p.Name)
	}

	// 获取所有角色
	var roles []struct {
		ID   primitive.ObjectID `bson:"_id"`
		Name string             `bson:"name"`
		Code string             `bson:"code"`
	}

	cursor, err = rolesCollection.Find(ctx, bson.M{})
	if err != nil {
		log.Fatal(err)
	}
	defer cursor.Close(ctx)

	if err := cursor.All(ctx, &roles); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("\nFound %d roles\n", len(roles))

	rand.Seed(time.Now().UnixNano())

	// 为每个角色随机分配权限
	for _, role := range roles {
		// 随机选择1到所有权限之间的数量
		numPermissions := rand.Intn(len(permissions)) + 1
		if numPermissions > len(permissions) {
			numPermissions = len(permissions)
		}

		// 随机选择权限
		selectedIndices := rand.Perm(len(permissions))[:numPermissions]
		var permissionIDs []primitive.ObjectID
		for _, idx := range selectedIndices {
			permissionIDs = append(permissionIDs, permissions[idx].ID)
		}

		// 将primitive.ObjectID转换为string
		permissionIDStrings := make([]string, len(permissionIDs))
		for i, id := range permissionIDs {
			permissionIDStrings[i] = id.Hex()
		}

		fmt.Printf("\nUpdating role '%s' (%s):\n", role.Name, role.Code)
		fmt.Printf("  Assigned %d permissions:\n", numPermissions)
		for _, idx := range selectedIndices {
			fmt.Printf("    - %s (%s)\n", permissions[idx].ID.Hex(), permissions[idx].Name)
		}

		// 更新角色文档
		update := bson.M{
			"$set": bson.M{
				"permission_ids": permissionIDStrings,
			},
		}

		result, err := rolesCollection.UpdateOne(
			ctx,
			bson.M{"_id": role.ID},
			update,
		)
		if err != nil {
			log.Printf("Error updating role %s: %v", role.Name, err)
			continue
		}

		fmt.Printf("  Updated %d document(s)\n", result.ModifiedCount)
	}

	fmt.Println("\n=== Script completed successfully ===")
}
