package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows"

	"golang.org/x/sys/windows/registry"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

type Task struct {
	TakeName  string
	Downloads []Download
	SubTasks  []SubTask
}

type Download struct {
	Url     string
	Version string
}

type SubTask struct {
	Command string
	Key     string
	Value   string
	/**
	追加模式中的插入位置-1 代表末尾
	*/
	index int
}

func main() {
	//HKEY_LOCAL_MACHINE\
	//key, err := registry.OpenKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Control\Session Manager\Environment\`, registry.ALL_ACCESS)
	//fmt.Println(err)
	//s, _, _ := key.GetStringValue("CLASSPATH1")
	//fmt.Println(s)
	//defer key.Close()
	//values, _ := key.ReadValueNames(0)
	//key.SetStringValue("String", "hello")
	//fmt.Println(values)

	//key.DeleteValue("String")
	jsonStr := `{
		    "takeName":"java",
		    "downloads":[
		        {
		            "url":"http:xxxxxxx",
		            "version":"1.8"
		        },        {
		            "url":"http:xxxxxxx",
		            "version":"1.7"
		        }
		    ],
		    "subTasks":[
		        {
		            "command":"env_cover",
		            "key":"JAVA_HOME",
		            "value":"C:\\Program Files (x86)\\Java\\jdk1.8.0_91"
		        },
		        {
		            "command":"env_append",
		            "key":"CLASSPATH1",
		            "value":"^%CLASSPATH1^%.;^%JAVA_HOME^%\\lib;^%JAVA_HOME^%\\lib\\dt.jar;^%JAVA_HOME^%\\lib\\tools.jar"
		        }
		    ]
		}`
	var javaTask Task
	json.Unmarshal([]byte(jsonStr), &javaTask)
	subTasks := javaTask.SubTasks
	for _, v := range subTasks {
		switch v.Command {
		case "env_cover":
			envCover(v)
		case "env_append":

			//cmdExecute("/c", "setx", v.Key, v.Value, "/m")
		}
	}
	unzip("1.zip", "test")
}

func envCover(task SubTask) {
	key, _ := registry.OpenKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Control\Session Manager\Environment\`, registry.ALL_ACCESS)
	defer key.Close()
	_, _, err := key.GetStringValue("tttt")
	if err == windows.ERROR_FILE_NOT_FOUND {
		fmt.Println("不存在")
	}

}
func unzip(zipFile string, destDir string) error {
	if !pathExists(zipFile) {
		return errors.New(`File "` + zipFile + `" not exist.`)
	}
	zipReader, err := zip.OpenReader(zipFile)
	if err != nil {
		return err
	}
	defer zipReader.Close()
	for _, f := range zipReader.File {
		decodeFileName := getUtf8FileName(f)
		filePath := filepath.Join(destDir, decodeFileName)
		if f.FileInfo().IsDir() {
			os.MkdirAll(filePath, os.ModePerm)
			continue
		}
		if err = os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
			return err
		}
		inFile, _ := f.Open()
		defer inFile.Close()
		outFile, _ := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		defer outFile.Close()
		io.Copy(outFile, inFile)

	}
	return nil
}

func getUtf8FileName(f *zip.File) string {
	decodeFileName := f.Name
	if f.NonUTF8 {
		decodeFileName = gbk2utf8([]byte(f.Name))
	}
	return decodeFileName
}

func gbk2utf8(data []byte) string {
	decoder := transform.NewReader(bytes.NewReader(data), simplifiedchinese.GB18030.NewDecoder())
	content, _ := ioutil.ReadAll(decoder)
	return string(content)
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return false
}
