package etcd_listen

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/blang/semver/v4"
	"github.com/joselee214/j7f/util"
	"sync"
	_ "time"
)

type WatchServerInfo struct {
	Name    string
	Version string
}
type WriteServerInfo struct {
	Serverkey string `json:"node_key"`
	Ip string	`json:"ip"`
	Port int	`json:"port"`
	//Value interface{}	`json:"value"`
}
type nodeInfo struct {
	NodeId string `json:"node_id"`
	Handles interface{} `json:"handles"`
	Address string `json:"address"`
	Port int `json:"port"`
	Metadata interface{} `json:"metadata"`
	Ip string `json:"ip"`
	Version string `json:"version"`
}

var Debugflag int = 1

type ListenerInputOutput interface {
	Export(string,map[string][]WriteServerInfo)
}

//输出 与 输入 的对应
var outputFileHasKeys = make(map[string]map[string]WatchServerInfo)
var lockOutputFileHasKeys sync.Mutex

// 关注的key  ->  服务名带版本 -> 分组文件
var namesWithOutputFile = make(map[string]map[string][]string)
var lockNamesWithOutputFile sync.Mutex

//输出结果 //key是分组 //
var outputFileMap = make( map[string]map[string][]WriteServerInfo )
var lockOutputFileMap sync.Mutex

func AddGroupWatchs(wsi map[string]WatchServerInfo,outputalias string) error  {
	//EmptyGroupWatchs(outputalias)
	if _,ok := outputFileHasKeys[outputalias]; ok {
		return errors.New( "already watched group ：" + outputalias )
	}

	//把配置文件的path加入观察队列
	lockOutputFileHasKeys.Lock()
	outputFileHasKeys[outputalias] = wsi
	lockOutputFileHasKeys.Unlock()

	lockNamesWithOutputFile.Lock()
	for sname, eachWatch := range wsi {
		if len(namesWithOutputFile[eachWatch.Name])==0 {
			namesWithOutputFile[eachWatch.Name] = make(map[string][]string)
		}
		namesWithOutputFile[eachWatch.Name][outputalias] = append(namesWithOutputFile[eachWatch.Name][outputalias],sname)
	}

	lockNamesWithOutputFile.Unlock()

	//debug("namesWithOutputFile222", outputalias,wsi)
	//debug("namesWithOutputFile222", namesWithOutputFile)

	lockOutputFileMap.Lock()
	if _,ok := outputFileMap[outputalias]; !ok {
		outputFileMap[outputalias] = make(map[string][]WriteServerInfo)
	}
	lockOutputFileMap.Unlock()
	return nil
}

//清空关注
func EmptyGroupWatchs(outputalias string) {

	lockOutputFileMap.Lock()
	if _,ok := outputFileMap[outputalias]; ok {
		delete(outputFileMap,outputalias)
	}
	lockOutputFileMap.Unlock()


	//debug("======== groupWatchs")
	if _,ok := outputFileHasKeys[outputalias]; ok {

		lockOutputFileHasKeys.Lock()
		groupWatchs := outputFileHasKeys[outputalias]
		delete(outputFileHasKeys,outputalias)
		lockOutputFileHasKeys.Unlock()

		lockNamesWithOutputFile.Lock()

		//debug("======== groupWatchs outputFileHasKeys outputalias")
		//debug(groupWatchs,outputalias)
		//debug("namesWithOutputFile",namesWithOutputFile)

		for sname , keyandversion := range groupWatchs { // /7YES_SERVICE/j7go_GRPC/^2.11:{/7YES_SERVICE/j7go_GRPC >=1.0.0}
			if v,ok1 := namesWithOutputFile[keyandversion.Name];ok1{
				if filekeyslist,ok3 := v[outputalias]; ok3{
					snameindex := util.Contains(filekeyslist,sname)
					if snameindex != -1 {
						namesWithOutputFile[keyandversion.Name][outputalias] = append(filekeyslist[:0], filekeyslist[snameindex+1:]...)
					}
				}
			}

			//debug("namesWithOutputFile111",sname , keyandversion)
			//debug("namesWithOutputFile111",namesWithOutputFile)

			if len(namesWithOutputFile[keyandversion.Name][outputalias]) == 0 {
				delete( namesWithOutputFile[keyandversion.Name],outputalias )
			}
			if len(namesWithOutputFile[keyandversion.Name]) == 0 {
				if etcdw.RemoveWatch( keyandversion.Name ) {
					debug( "Etcd removeWatch : ",keyandversion.Name )
				}
				delete( namesWithOutputFile,keyandversion.Name )
			}
		}

		lockNamesWithOutputFile.Unlock()

		//debug( " === end namesWithOutputFile" )
		//debug( etcdw )

	} else {
		//debug("======== outputFileHasKeys")
		//debug(outputFileHasKeys)
	}

}


var lsio  ListenerInputOutput
var etcdw *EtcdWatcher

func Start(etcdservers []string,listenIo ListenerInputOutput) error {
	lsio = listenIo


	//debug("==============start namesWithOutputFile",namesWithOutputFile)
	//debug("==============start outputFileHasKeys",outputFileHasKeys)
	//debug("==============start outputFileMap",outputFileMap)

	debug("==============start")

	ew, err := NewEtcdWatcher(etcdservers)
	etcdw = ew
	if err == nil {
		var wg sync.WaitGroup
		ReloadEtcdWatch()
		wg.Add(1)
		wg.Wait()
	} else {
		debug(err)
	}
	return nil
}

func ReloadEtcdWatch() {
	for key ,_ := range namesWithOutputFile {
		if etcdw != nil {
			if etcdw.AddWatch(key,true,lsc) {
				debug("Etcd addWatchKey:",key)
			}
		}
	}
}

func ReloadExport() {
	for key ,_ := range namesWithOutputFile {
		tiggerChangesWatchKey(key)
	}
}

func test() {
	//time.Sleep( 5 * time.Second )
	//EmptyGroupWatchs("grpc1.json")
}

func debug(a ...interface{}){
	if Debugflag == 1 {
		//var xx = make([]interface{},1)
		//xx[0] = "etcd_listen debug ::: "
		a = append( []interface{}{"etcd_listen debug ::: "},a...)
		fmt.Println( a... )
	}
}

var lockEtcd sync.Mutex //互斥锁
var etcdSettings = make(map[string]map[string]nodeInfo)

//关注key变化
func tiggerChangeWatchKeyValue(watchkey,key,value string)  {

	lockEtcd.Lock()
	defer lockEtcd.Unlock()
	if( key=="" ){
		if _,ok := etcdSettings[watchkey] ; ok{
			delete(etcdSettings,watchkey)
		}
		return
	}
	if( value=="" ){
		if _,ok := etcdSettings[watchkey] ; ok{
			if _,ok := etcdSettings[watchkey][key] ; ok{
				delete(etcdSettings[watchkey],key)
				if len(etcdSettings[watchkey])== 0 {
					delete(etcdSettings,watchkey)
				}
			}
		}
		return
	}

	if _,ok := etcdSettings[watchkey] ; !ok{
		etcdSettings[watchkey] = make(map[string]nodeInfo)
	}

	var ni nodeInfo
	error := json.Unmarshal( []byte(value) , &ni )
	if error == nil {
		etcdSettings[watchkey][key] = ni
	} else {
		debug(error)
	}
}

//刷新key变化
func tiggerChangesWatchKey(watchkey string)  {
	if _,ok := namesWithOutputFile[watchkey] ; ok{
		for eachfile,namekeys := range namesWithOutputFile[watchkey]{
			lockOutputFileMap.Lock()

			//debug( "--------------------- Export namekeys namekeys namekeys" )
			//debug( watchkey,eachfile,namekeys )

			if _,ok := outputFileMap[eachfile]; !ok {
				outputFileMap[eachfile] = make(map[string][]WriteServerInfo)
			}

			for _,namekey := range namekeys {


				if _,ok:= outputFileHasKeys[eachfile]; ok { //存在关注文件
					if fconfig,okk:= outputFileHasKeys[eachfile][namekey]; okk { //存在关注key

						//debug( " 1--------------------- Export namekey " )
						//debug( watchkey,eachfile,namekey,outputFileMap )

						delete(outputFileMap[eachfile],namekey)
						if len(outputFileMap[eachfile]) == 0 {
							outputFileMap[eachfile] = make(map[string][]WriteServerInfo)
						}
						outputFileMap[eachfile][namekey] = make([]WriteServerInfo, 0)

						if servernodes,okkk := etcdSettings[fconfig.Name]; okkk {
							//debug( "servernodes",servernodes )
							for nodeallname,nodeInfo := range servernodes {
								vA,errsemver := semver.Parse(nodeInfo.Version)
								if errsemver==nil {
									expectedRange, errsemver1 := semver.ParseRange(fconfig.Version)
									if errsemver1==nil{
										if expectedRange(vA) {
											//符合 semver
											outputFileMap[eachfile][namekey] = append(outputFileMap[eachfile][namekey],WriteServerInfo{
												Serverkey: nodeallname,
												Ip:        nodeInfo.Ip,
												Port:      nodeInfo.Port,
												//Value:     nodeInfo.Handles,
											})
										}
									}
								}
							}
						} else {
							debug( "None server nodes : " ,eachfile,namekey )
						}
					}
				}
			}

			lsio.Export(eachfile,outputFileMap[eachfile])

			//debug( "Export outputFileMap outputFileMap outputFileMap outputFileMap" , eachfile )
			//debug( outputFileMap )

			lockOutputFileMap.Unlock()
		}
	}
}

type ListenerSC struct {
}

var lsc = &ListenerSC{}

func (lsc *ListenerSC) Set(watchkey string, key []byte, value []byte) {
	tiggerChangeWatchKeyValue(watchkey,string(key),string(value))
}
func (lsc *ListenerSC) SetOk(watchkey string) {
	tiggerChangesWatchKey(watchkey)
}

func (lsc *ListenerSC) Create(watchkey string, key []byte, value []byte) {
	tiggerChangeWatchKeyValue(watchkey,string(key),string(value))
	tiggerChangesWatchKey(watchkey)
}
func (lsc *ListenerSC) Modify(watchkey string, key []byte,value []byte) {
	tiggerChangeWatchKeyValue(watchkey,string(key),string(value))
	tiggerChangesWatchKey(watchkey)
}
func (lsc *ListenerSC) Delete(watchkey string, key []byte) {
	tiggerChangeWatchKeyValue(watchkey,string(key),"")
	tiggerChangesWatchKey(watchkey)
}
func (lsc *ListenerSC) Empty(watchkey string) {
	tiggerChangeWatchKeyValue(watchkey,"","")
	tiggerChangesWatchKey(watchkey)
}
