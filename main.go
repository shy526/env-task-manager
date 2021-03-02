package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows"

	"github.com/beevik/etree"
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
	Urls    []string
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
	XPath string
}

func main() {
	downloadPath := "D:\\git_weidian\\env-task-manager"
	jsonStr := `{
		    "takeName":"java",
			"destDir":"java8",
		    "downloads":[
		        {
		            "urls":[
					"https://codechina.csdn.net/qq_19763819/devlop-env-zip/-/raw/master/java/jdk1.8.0_281.zip?env_fileName=jdk1.8.0_281",
					"https://codechina.csdn.net/qq_19763819/devlop-env-zip/-/raw/master/java/jre1.8.0_281.zip?env_fileName=jre1.8.0_281"
					],
		            "version":"1.8"
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
		            "command":"xml_append",
		            "key":"D:\\javaEx\\apache-maven-3.6.3\\conf\\settings.xml",
		            "value":"<localRepository>xxxxxx</localRepository>",
		            "xPath":"./settings"
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
	os.Setenv("env_task_downloadPath", downloadPath)
	fmt.Println(rootDir)

	/*	for _, v := range downLoadTask(javaTask.TakeName, javaTask.Downloads[0]) {
		unzipTask(v, rootDir)
	}*/

	//

	//执行配置
	subTasks := javaTask.SubTasks
	for _, v := range subTasks {
		v.Value = os.ExpandEnv(v.Value)
		switch v.Command {
		case "env_cover":
			//sourceStr := envCoverTask(v)
			//fmt.Printf("%s --覆盖-> %s \n", sourceStr, v.Value)
		case "env_append":
		//sourceStr, appendStr := envAppendTask(v)
		//fmt.Printf("%s\n --追加-> \n%s\n ==\n%s\n", sourceStr, v.Value, appendStr)
		case "xml_append":
			xmlUpdateValue(v)
		}
	}
}

/**
覆盖
*/
func envCoverTask(task SubTask) string {
	key, _ := registry.OpenKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Control\Session Manager\Environment\`, registry.ALL_ACCESS)
	defer key.Close()
	source, _, _ := key.GetStringValue(task.Key)
	key.SetExpandStringValue(task.Key, task.Value)
	return source
}

/**
追加
*/
func envAppendTask(task SubTask) (string, string) {
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

func unzipTask(zipFile string, destDir string) (string, error) {
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
		io.Copy(outFile, io.TeeReader(inFile, &WriteCounter{Total: uint64(f.FileInfo().Size()), path: filePath}))
	}
	fmt.Println()
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

type WriteCounter struct {
	Total uint64
	path  string
	Index uint64
}

func (wc *WriteCounter) Write(p []byte) (int, error) {
	n := len(p)
	wc.Index += uint64(n)
	wc.PrintProgress()
	return n, nil
}

func (wc WriteCounter) PrintProgress() {
	fmt.Printf("%s --> %v/%v \r", wc.path, wc.Index, wc.Total)
}
func downLoadTask(taskName string, download Download) []string {
	var result []string
	for _, v := range download.Urls {
		req, _ := http.NewRequest("GET", v, nil)
		fileName := req.URL.Query().Get("env_fileName")
		downloadPath := taskName + "_" + fileName + ".zip"
		result = append(result, downloadPath)
		out, _ := os.Create(downloadPath)
		defer out.Close()
		resp, _ := http.Get(v)
		defer resp.Body.Close()
		io.Copy(out, io.TeeReader(resp.Body, &WriteCounter{Total: uint64(resp.ContentLength), path: downloadPath}))
		fmt.Println()
	}
	return result
}

func xmlUpdateValue(task SubTask) {
	path := task.Key
	fileName := filepath.Base(path)
	doc := etree.NewDocument()
	doc.ReadFromFile(path)
	el := doc.FindElement(task.XPath)
	el.AddChild(crateElementFromStr(task.Value))
	doc.Indent(4)
	bakPath := filepath.Join(filepath.Dir(path), "dev_task_"+fileName)
	os.Rename(path, bakPath)
	doc.WriteToFile(path)

}

func crateElementFromStr(elementStr string) *etree.Element {
	doc := etree.NewDocument()
	doc.ReadFromString(elementStr)
	return doc.ChildElements()[0].Copy()
}
