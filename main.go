package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows"

	"golang.org/x/sys/windows/registry"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

type Task struct {
	TakeName  string
	DestDir   string
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
	downloadPath := "D:\\javaEx"
	jsonStr := `{
		    "takeName":"java",
			"destDir":"java8",
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
		            "value":"$env_task_rootDir\\jdk1.8.0_281"
		        },
		        {
		            "command":"env_append",
		            "key":"CLASSPATH",
		            "value":".;%JAVA_HOME%\\lib;%JAVA_HOME%\\lib\\dt.jar;%JAVA_HOME%\\lib\\tools.jar"
		        },
				{
					"command":"env_append",
					"key":"path",
					"value":"%JAVA_HOME%\\bin"
				}
		    ]
		}`
	var javaTask Task
	json.Unmarshal([]byte(jsonStr), &javaTask)
	rootDir := filepath.Join(downloadPath, javaTask.DestDir)
	os.Setenv("env_task_rootDir", rootDir)
	fmt.Println(rootDir)
	unzip("jdk8u281.zip", rootDir)

	//执行配置
	subTasks := javaTask.SubTasks
	for _, v := range subTasks {
		v.Value = os.ExpandEnv(v.Value)
		switch v.Command {
		case "env_cover":
			sourceStr := envCover(v)
			fmt.Printf("%s --覆盖-> %s \n", sourceStr, v.Value)
		case "env_append":
			sourceStr, appendStr := envAppend(v)
			fmt.Printf("%s\n --追加-> \n%s\n ==\n%s\n", sourceStr, v.Value, appendStr)
		}
	}
}

/**
覆盖
*/
func envCover(task SubTask) string {
	key, _ := registry.OpenKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Control\Session Manager\Environment\`, registry.ALL_ACCESS)
	defer key.Close()
	source, _, _ := key.GetStringValue(task.Key)
	key.SetExpandStringValue(task.Key, task.Value)
	return source
}

/**
追加
*/
func envAppend(task SubTask) (string, string) {
	key, _ := registry.OpenKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Control\Session Manager\Environment\`, registry.ALL_ACCESS)
	defer key.Close()
	sourceStr, _, err := key.GetStringValue(task.Key)
	if err == windows.ERROR_FILE_NOT_FOUND {
		key.SetExpandStringValue(task.Key, task.Value)
	} else {
		key.SetExpandStringValue(task.Key, distinctStr(task.Value, sourceStr))
	}
	appendStr, _, _ := key.GetStringValue(task.Key)
	return sourceStr, appendStr
}

func unzip(zipFile string, destDir string) (string, error) {
	result := ""
	zipReader, err := zip.OpenReader(zipFile)
	if err != nil {
		return result, err
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
			return result, err
		}
		inFile, _ := f.Open()
		defer inFile.Close()
		outFile, _ := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		defer outFile.Close()
		io.Copy(outFile, inFile)

	}
	return result, nil
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

func distinctStr(append string, source string) string {
	set := make(map[string]byte)
	var result []string
	result = distinct(strings.Split(append, ";"), set, result)
	result = distinct(strings.Split(source, ";"), set, result)
	fmt.Println(strings.Join(result, ";"))
	return strings.Join(result, ";")
}

func distinct(arrayA []string, set map[string]byte, result []string) []string {
	for _, item := range arrayA {
		if _, ok := set[item]; !ok {
			set[item] = 1
			result = append(result, item)
		}
	}
	return result
}
