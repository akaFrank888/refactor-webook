package mongodb

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"testing"
	"time"
)

type MongoDBTestSuite struct {
	suite.Suite
	collection *mongo.Collection
}

func (s *MongoDBTestSuite) SetupSuite() {
	// 一、初始化客户端的过程
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	monitor := &event.CommandMonitor{
		Started: func(ctx context.Context, evt *event.CommandStartedEvent) {
			fmt.Println(evt.Command)
		},
	}
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://root:example@localhost:27017/").SetMonitor(monitor))
	assert.NoError(s.T(), err)
	collection := client.Database("webook").Collection("articles")
	s.collection = collection
	// 三、插入一些准备数据
	manyRes, err := collection.InsertMany(ctx, []any{
		Article{
			Id:       12,
			AuthorId: 100,
		},
		Article{
			Id:       13,
			AuthorId: 200,
		},
	})
	assert.NoError(s.T(), err)
	s.T().Log("插入数量", len(manyRes.InsertedIDs))

}

func (s *MongoDBTestSuite) TearDownSuite() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err := s.collection.DeleteMany(ctx, bson.D{})
	assert.NoError(s.T(), err)
	_, err = s.collection.Indexes().DropAll(ctx)
	assert.NoError(s.T(), err)
}

func (s *MongoDBTestSuite) TestOr() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	filter := bson.A{bson.D{bson.E{"id", 12}},
		bson.D{bson.E{"id", 13}}}
	res, err := s.collection.Find(ctx, bson.D{bson.E{"$or", filter}})
	assert.NoError(s.T(), err)
	var arts []Article
	err = res.All(ctx, &arts)
	assert.NoError(s.T(), err)
	s.T().Log("查询结果", arts)
}

func (s *MongoDBTestSuite) TestAnd() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	filter := bson.A{bson.D{bson.E{"id", 12}},
		bson.D{bson.E{"author_id", 100}}}
	res, err := s.collection.Find(ctx, bson.D{bson.E{"$and", filter}})
	assert.NoError(s.T(), err)
	var arts []Article
	err = res.All(ctx, &arts)
	assert.NoError(s.T(), err)
	s.T().Log("查询结果", arts)
}

func (s *MongoDBTestSuite) TestIn() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	filter := bson.D{bson.E{"id",
		bson.D{bson.E{"$in", []int{12, 13}}}}}
	//proj := bson.D{bson.E{"id", 1}}
	proj := bson.M{"id": 1}
	res, err := s.collection.Find(ctx, filter,
		// 查询特定字段
		options.Find().SetProjection(proj))
	assert.NoError(s.T(), err)
	var arts []Article
	err = res.All(ctx, &arts)
	assert.NoError(s.T(), err)
	s.T().Log("查询结果", arts)
}

func (s *MongoDBTestSuite) TestIndexes() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	ires, err := s.collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		// 将字段 id 设为索引（经常用id来查询）
		Keys:    bson.D{bson.E{"id", 1}},
		Options: options.Index().SetUnique(true).SetName("idx_id"),
	})
	assert.NoError(s.T(), err)
	s.T().Log("创建索引", ires)
}

func TestMongoDBQueries(t *testing.T) {
	suite.Run(t, &MongoDBTestSuite{})
}
