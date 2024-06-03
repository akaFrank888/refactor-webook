package mongodb

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/event"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestMongoDB(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	monitor := &event.CommandMonitor{
		Started: func(ctx context.Context, evt *event.CommandStartedEvent) {
			fmt.Println(evt.Command)
		},
	}

	// 通过 Connect 获得客户端
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://root:example@localhost:27017/").SetMonitor(monitor))

	assert.NoError(t, err)
	// 客户端调用 Database 和 Collection 获得数据库和集合
	// collection 用于增删改查
	collection := client.Database("webook").Collection("articles")

	// （一）插入文档
	// note 为被插入document分配的_id
	res, err := collection.InsertOne(ctx, Article{
		// note MongoDB 没有自增主键的说法，所以要手动写入 Id
		Id:       1,
		AuthorId: 123,
		Title:    "我的标题",
		Content:  "我的内容",
	})
	oid := res.InsertedID.(primitive.ObjectID)
	t.Log("插入文档的id；", oid)

	// （二）查询文档
	// note 利用 bson 构造查询条件
	// queryFilter := bson.D{bson.E{Key: "id", Value: 1}}
	queryFilter := bson.M{"id": 1}
	findRes := collection.FindOne(ctx, queryFilter)
	if findRes.Err() == mongo.ErrNoDocuments {
		t.Log("没有找到文档")
	} else {
		var article Article
		err = findRes.Decode(&article)
		t.Log(article)
	}

	// （三）更新文档
	updateFilter := bson.M{"id": 1}
	//set := bson.D{bson.E{Key: "$set", Value: bson.E{Key: "title", Value: "新的标题"}}}
	set := bson.D{bson.E{Key: "$set", Value: bson.M{
		"title": "我的标题",
	}}}
	// note 如果是要更新多个文档，就调用 UpdateMany
	updateOneRes, err := collection.UpdateOne(ctx, updateFilter, set)
	t.Log("更新文档数量", updateOneRes.ModifiedCount)

	// （四）删除文档
	deleteFilter := bson.M{"id": 1}
	deleteOneRes, err := collection.DeleteMany(ctx, deleteFilter)
	t.Log("删除文档数量", deleteOneRes.DeletedCount)

}

type Article struct {
	Id       int64  `bson:"id,omitempty"`
	Title    string `bson:"title,omitempty"`
	Content  string `bson:"content,omitempty"`
	AuthorId int64  `bson:"author_id,omitempty"`
	Status   uint8  `bson:"status,omitempty"`
	Ctime    int64  `bson:"ctime,omitempty"`
	// 更新时间
	Utime int64 `bson:"utime,omitempty"`
}
