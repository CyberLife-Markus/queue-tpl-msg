package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	wechat "queue/tasks"
	tracers "queue/tracers"

	"github.com/RichardKnop/machinery/v1"
	"github.com/RichardKnop/machinery/v1/config"
	"github.com/RichardKnop/machinery/v1/log"
	"github.com/RichardKnop/machinery/v1/tasks"
	"github.com/urfave/cli"

	// ORM
	"github.com/gohouse/gorose"
	_ "github.com/gohouse/gorose/driver/mysql"
)

var (
	app        *cli.App
	configPath string
)

var dbConfig = &gorose.DbConfigSingle{
	Driver:          os.Getenv("DB_TYPE"),   // 驱动: mysql/sqlite/oracle/mssql/postgres
	EnableQueryLog:  false,                  // 是否开启sql日志
	SetMaxOpenConns: 0,                      // (连接池)最大打开的连接数，默认值为0表示不限制
	SetMaxIdleConns: 0,                      // (连接池)闲置的连接数
	Prefix:          os.Getenv("DB_PREFIX"), // 表前缀
	// 数据库链接
	Dsn: os.Getenv("DB_USER") +
		":" + os.Getenv("DB_PASS") +
		"@tcp(" + os.Getenv("DB_HOST") +
		":" + os.Getenv("DB_PORT") +
		")/" + os.Getenv("DB_NAME") +
		"?charset=" + os.Getenv("DB_CHARSET"),
}

func init() {
	// Initialise a CLI app
	app = cli.NewApp()
	app.Name = "Queue"
	app.Usage = "Wechat template message push message queue"
	app.Author = "Markus"
	app.Email = "i@yoyoyo.me"
	app.Version = "1.0.0"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "c",
			Value:       "",
			Destination: &configPath,
			Usage:       "Path to a configuration file",
		},
	}
}

func main() {
	// Set the CLI app commands
	app.Commands = []cli.Command{
		{
			Name:  "worker",
			Usage: "Start worker server",
			Action: func(c *cli.Context) error {
				if err := worker(); err != nil {
					return cli.NewExitError(err.Error(), 1)
				}
				return nil
			},
		}, {
			Name:  "httpServer",
			Usage: "Start http server",
			Action: func(c *cli.Context) {
				httpServer()
			},
		},
	}

	// Run the CLI app
	app.Run(os.Args)
}

func loadConfig() (*config.Config, error) {
	if configPath != "" {
		return config.NewFromYaml(configPath, true)
	}

	return config.NewFromEnvironment(true)
}

func startServer() (*machinery.Server, error) {
	cnf, err := loadConfig()
	if err != nil {
		return nil, err
	}

	// Create server instance
	server, err := machinery.NewServer(cnf)
	if err != nil {
		return nil, err
	}

	// Register tasks
	job := map[string]interface{}{
		"pushTplMsg": wechat.PushTplMsg,
	}

	return server, server.RegisterTasks(job)
}

/**
 * 任务入列
 */
func pushJob(data []map[string]interface{}, employer string, time string, url string) error {
	cleanup, err := tracers.SetupTracer("sender")
	if err != nil {
		log.FATAL.Fatalln("Unable to instantiate a tracer:", err)
	}
	defer cleanup()

	server, err := startServer()
	if err != nil {
		return err
	}

	// 创建任务
	pushTplMsg := make([]*tasks.Signature, len(data))

	for k, v := range data {
		t := tasks.Signature{
			Name: "pushTplMsg",
			Args: []tasks.Arg{
				{
					Type:  "[]string",
					Value: []string{v["openid"].(string), employer, time, url},
				},
			},
		}
		pushTplMsg[k] = &t
	}

	group, err := tasks.NewGroup(pushTplMsg...)

	// 推送入列
	asyncResults, err := server.SendGroup(group, 0)

	if err != nil {
		return fmt.Errorf("Could not send group: %s", err.Error())
	}

	log.INFO.Printf("asyncResults: %v\n", asyncResults)

	return nil
}

/**
 * 获取用户Openid
 */
func queryData(w http.ResponseWriter, r *http.Request) {
	connection, err := gorose.Open(dbConfig)

	if err != nil {
		fmt.Println(err.Error())
		return
	}

	//新建会话
	db := connection.NewSession()

	// 判断用户列表还是分类
	var param []string
	param = strings.Split(r.PostFormValue("data"), ",")

	if r.PostFormValue("type") == "1" {
		query, err := db.Table("wechat").
			Limit(len(param)).
			WhereIn("uuid", param).
			Fields("openid").
			Get()

		if err != nil {
			fmt.Println(err.Error())
			return
		}

		// 推送队列
		go func() {
			_ = pushJob(query, r.PostFormValue("employer"), r.PostFormValue("time"), r.PostFormValue("url"))
		}()

		_, err = w.Write([]byte("success"))

		if err != nil {
			fmt.Println(err.Error())
			return
		}

		return
	} else {
		var where string

		query := db.Table("wechat a").
			Join("db_member b", "a.uuid", "=", "b.uuid").
			LeftJoin("db_authentication c", "a.uuid", "=", "c.uuid").
			Fields("a.openid")

		for _, v := range param {
			switch v {
			// 女主播
			case "0":
				if len(where) > 2 {
					where += " or b.`sex` = 1"
				} else {
					where += "b.`sex` = 1"
				}
				break

			//男主播
			case "1":
				if len(where) > 2 {
					where += " or b.`sex` = 2"
				} else {
					where += "b.`sex` = 2"
				}
				break

			//签约认证通过
			case "2":
				if len(where) > 2 {
					where += " or c.`type` = 1"
				} else {
					where += "c.`type` = 1"
				}
				break

			//提交签约未认证
			case "3":
				if len(where) > 2 {
					where += " or c.`type` in (0,2)"
				} else {
					where += "c.`type` in (0,2)"
				}
				break
			//已勾选接受试音通知的
			case "4":
				if len(where) > 2 {
					where += " or b.`audition_notice` = 1"
				} else {
					where += "b.`audition_notice` = 1"
				}
				break
			default:
			}
		}

		data, err := query.Where("(b.`lock` = 0) AND (" + where + ")").Get()

		if err != nil {
			fmt.Println(err.Error())
			return
		}

		// 推送队列
		go func() {
			_ = pushJob(data, r.PostFormValue("employer"), r.PostFormValue("time"), r.PostFormValue("url"))
		}()

		_, err = w.Write([]byte("success"))

		if err != nil {
			fmt.Println(err.Error())
			return
		}

		return
	}
}

/**
 * HTTP API
 */
func httpServer() {

	// Http server心跳检测
	http.HandleFunc("/heartbeat", func(w http.ResponseWriter, r *http.Request) {

	})

	// 任务入列 API
	http.HandleFunc("/pushJob", queryData)

	log.INFO.Printf("Success Http server is running!Port is : " + os.Getenv("HTTP"))

	err := http.ListenAndServe("0.0.0.0:"+os.Getenv("HTTP"), nil)

	if err != nil {
		log.WARNING.Printf("WARNING Http server start failed: %s", err)
	}
}

/**
 * 任务消费
 */
func worker() error {
	consumerTag := "queue_worker"

	cleanup, err := tracers.SetupTracer(consumerTag)
	if err != nil {
		log.FATAL.Fatalln("Unable to instantiate a tracer:", err)
	}
	defer cleanup()

	server, err := startServer()
	if err != nil {
		return err
	}

	// The second argument is a consumer tag
	// Ideally, each worker should have a unique tag (worker1, worker2 etc)
	worker := server.NewWorker(consumerTag, 0)

	// Here we inject some custom code for error handling,
	// start and end of task hooks, useful for metrics for example.
	errorhandler := func(err error) {
		log.ERROR.Println("I am an error handler:", err)
	}

	pretaskhandler := func(signature *tasks.Signature) {
		//log.INFO.Println("I am a start of task handler for:", signature.Name)
	}

	posttaskhandler := func(signature *tasks.Signature) {
		//log.INFO.Println("I am an end of task handler for:", signature.Name)
	}

	worker.SetPostTaskHandler(posttaskhandler)
	worker.SetErrorHandler(errorhandler)
	worker.SetPreTaskHandler(pretaskhandler)

	return worker.Launch()
}
