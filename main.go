// camup project main.go
package main

import (
	"encoding/json"

	"golang.org/x/crypto/ssh"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"log"
	"os"
	"path/filepath"
)

//Структура екоторую мы забираем из конфига
type Config struct {
	Server string
	Cams   []Cams
}

type Cams struct {
	Name string
	Port string
}

//Конец структуры

//Структура для забирания данных из монги и работы функций
type Cam struct {
	Name  string
	Host  string
	State string
}

//Функция соединения по ssh и выполнения команды на сервере
func executeCmd(cmd, hostname string, port string, name string) string {

	sshconf := &ssh.ClientConfig{
		User: "ubnt",
		Auth: []ssh.AuthMethod{ssh.Password("ubnt")},
	}
	conn, err := ssh.Dial("tcp", hostname+":"+port, sshconf)
	if err != nil {
		//log.Println(hostname, ":", port, " ", err)
		return name + " " + err.Error()

	}
	session, err := conn.NewSession()
	if err != nil {
		//log.Println(err)
		return name + " " + err.Error()
	}
	defer session.Close()

	//var stdoutBuf bytes.Buffer
	//session.Stdout = &stdoutBuf
	if err := session.Run(cmd); err != nil {
		//log.Println(err)
		return "Камера " + name + " ошибка подключения: " + err.Error()
	}
	return "Камера " + name + " подключена"

}

//функция проверки дисконект камеры из конфига
func camStatus(name string) string {

	session, err := mgo.Dial("localhost:7441")
	if err != nil {
		log.Println(err)
	}
	defer session.Close()
	result := Cam{}
	c := session.DB("av").C("camera")
	err = c.Find(bson.M{"name": name}).One(&result)
	if err != nil {
		log.Println(name, err)
	}
	if result.State == "DISCONNECTED" {
		return result.Host
	} else {
		return ""
	}
}

func main() {

	// Будем писать все логи в файл camup.log
	progRoot, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		panic(err)
	}

	logFile, err := os.OpenFile(progRoot+"/camup.log", os.O_RDWR|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}

	defer logFile.Close()
	log.SetOutput(logFile)

	//Открываем конфиг, где указывается какие камеры нужно проверять
	file, err := os.OpenFile(progRoot+"/camup.cfg", os.O_RDWR|os.O_APPEND, 0666)
	if err != nil {
		log.Println(err)
		return
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	config := new(Config)
	err1 := decoder.Decode(&config)
	if err1 != nil {
		log.Println(err1)
	}

	//Что выполняем на удаленном хосте
	cmd := "sed -i '/unifivideo.server/d' /tmp/system.cfg; echo unifivideo.server='" + config.Server + "' >> /tmp/system.cfg;cfgmtd -w -p /etc/; reboot "
	//cmd := "ifconfig"

	messages := make(chan string, 10)
	y := 0
	for _, camname := range config.Cams {
		camstat := camStatus(camname.Name)
		camport := camname.Port
		camNname := camname.Name

		if camstat != "" {
			y++
			go func(camstat string) {

				messages <- executeCmd(cmd, camstat, camport, camNname)

			}(camstat)
			//log.Println("Камера ", camname.Name, <-messages)
		}

	}

	for i := 0; i < y; i++ {

		res := <-messages
		log.Println(res)

	}

}
