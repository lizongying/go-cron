package app

import (
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"log"
)

var MongoDatabase *mongo.Database

func init() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
}

func InitMongo(mongoParam *Mongo) {
	client, err := mongo.Connect(Ctx, options.Client().ApplyURI(mongoParam.Uri))
	if err != nil {
		log.Fatalln(err)
	}

	//defer func() {
	//	if err = client.Disconnect(Ctx); err != nil {
	//		log.Println(err)
	//	}
	//}()

	err = client.Ping(Ctx, readpref.Primary())
	if err != nil {
		log.Fatalln(err)
	}

	MongoDatabase = client.Database(mongoParam.Database)
}
