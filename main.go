package main

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"github.com/robfig/cron/v3"
	"go-cron/app"
	"go.mongodb.org/mongo-driver/bson"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

type Task struct {
	Cmd     *Cmd
	EntryID cron.EntryID
	Md5     string
	State   string
}
type Cmd struct {
	Id     int
	Script string
	Dir    string
	Spec   string
	Group  string
	Enable bool
}

var TaskMap = make(map[int]*Task, 0)

var (
	Info    *log.Logger
	Warning *log.Logger
	Error   *log.Logger
)

var Interval = time.Minute
var Group = ""
var CronFile = ""
var Collection = ""

func init() {
	app.InitConfig()
	mongo := app.Conf.Mongo
	app.InitMongo(mongo)
	CronFile = app.Conf.CronFile
	Group = app.Conf.Group
	Collection = mongo.Collection
	Interval = time.Duration(app.Conf.Interval) * time.Minute
	logFile, err := os.OpenFile(app.Conf.Log.Filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalln("open log file failed")
	}
	Info = log.New(os.Stdout, "Info:", log.Ldate|log.Ltime|log.Lshortfile)
	Warning = log.New(os.Stdout, "Warning:", log.Ldate|log.Ltime|log.Lshortfile)
	Error = log.New(io.MultiWriter(os.Stderr, logFile), "Error:", log.Ldate|log.Ltime|log.Lshortfile)
}

func main() {
	run()
	select {}
}
func run() {
	c := cron.New()
	c.Start()
	for {
		var confList []struct {
			Id     int    `json:"id"`
			Script string `json:"script"`
			Dir    string `json:"dir"`
			Spec   string `json:"spec"`
			Group  string `json:"group"`
			Enable bool   `json:"enable"`
		}
		var cmdList []*Cmd
		if CronFile != "" {
			conf, err := ioutil.ReadFile(CronFile)
			if err != nil {
				Error.Println("read config file failed")
				time.Sleep(Interval)
				continue
			}

			if err := json.Unmarshal(conf, &confList); err != nil {
				Error.Println("parse config file failed")
				time.Sleep(Interval)
				continue
			}
			for _, cmd := range confList {
				cmdList = append(cmdList, &Cmd{
					Id:     cmd.Id,
					Script: cmd.Script,
					Dir:    cmd.Dir,
					Spec:   cmd.Spec,
					Group:  cmd.Group,
					Enable: cmd.Enable,
				})
			}
		} else {
			cur, err := app.MongoDatabase.Collection(Collection).Find(app.Ctx, bson.D{{}})
			if err != nil {
			}
			for cur.Next(app.Ctx) {
				var cmd Cmd
				err := cur.Decode(&cmd)
				if err != nil {
					Info.Println(err)
					continue
				}
				cmdList = append(cmdList, &Cmd{
					Id:     cmd.Id,
					Script: cmd.Script,
					Dir:    cmd.Dir,
					Spec:   cmd.Spec,
					Group:  cmd.Group,
					Enable: cmd.Enable,
				})
			}
		}

		for _, cmd := range cmdList {
			go func(cmd *Cmd) {
				taskId := cmd.Id
				script := cmd.Script
				dir := cmd.Dir
				spec := cmd.Spec
				group := cmd.Group
				enable := cmd.Enable
				if group != Group || !enable {
					//删除任务
					if TaskMap[taskId] == nil {
						//Info.Println("cmd is not in cron:", *cmd)
						return
					}
					entryID := TaskMap[taskId].EntryID
					if entryID == 0 {
						//Info.Println("cmd is not in cron:", cmd)
						return
					}
					c.Remove(entryID)
					delete(TaskMap, taskId)
					Info.Println("remove cmd from cron:", *cmd)
					return
				}
				if TaskMap[taskId] == nil {
					TaskMap[taskId] = &Task{}
				}
				taskMd5 := fmt.Sprintf("%x", md5.Sum([]byte(script+dir+spec)))
				entryID := TaskMap[taskId].EntryID
				if entryID > 0 {
					//Info.Println("cmd is in cron:", cmd)
					//修改任务
					if TaskMap[taskId].Md5 != taskMd5 {
						entryIDOld := entryID
						cmdOld := cmd
						entryID, _ = c.AddFunc(spec, func() {
							execScript(*cmd)
						})
						if entryID == 0 {
							Info.Println("add cmd failed:", *cmd)
							return
						}
						TaskMap[taskId].Md5 = taskMd5
						TaskMap[taskId].EntryID = entryID
						TaskMap[taskId].Cmd = cmd
						Info.Println("add cmd from cron:", *cmd)
						c.Remove(entryIDOld)
						Info.Println("remove cmd from cron:", *cmdOld)
					}
					return
				}
				//增加任务
				entryID, _ = c.AddFunc(spec, func() {
					execScript(*cmd)
				})
				if entryID == 0 {
					delete(TaskMap, taskId)
					Info.Println("add cmd failed:", *cmd)
					return
				}
				TaskMap[taskId].Md5 = taskMd5
				TaskMap[taskId].EntryID = entryID
				TaskMap[taskId].Cmd = cmd
				Info.Println("add cmd to cron:", *cmd)
			}(cmd)
		}
		time.Sleep(Interval)
	}
}

func execScript(cmd Cmd) {
	taskId := cmd.Id
	script := cmd.Script
	dir := cmd.Dir
	if TaskMap[taskId].State == "RUN" {
		Info.Println("cmd is in process:", cmd)
		return
	}
	s := strings.Split(script, " ")
	shell := exec.Command(s[0], s[1:]...)
	if dir != "" {
		shell.Dir = dir
	}
	err := shell.Start()
	if err != nil {
		Error.Println("cmd run failed:", cmd)
		return
	}
	go func() {
		if err := shell.Wait(); err != nil {
			Info.Println("cmd is killed:", cmd)
			TaskMap[taskId].State = "DIE"
			return
		}
		Info.Println("cmd is finished:", cmd)
		TaskMap[taskId].State = "FIN"
	}()
	TaskMap[taskId].State = "RUN"
	Info.Println(shell.Process.Pid, shell)
}
