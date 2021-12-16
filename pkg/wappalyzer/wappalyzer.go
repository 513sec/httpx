package wappalyzer

import (
	"embed"
	"encoding/json"
	"fmt"
	"github.com/bufsnake/httpx/pkg/log"
	"io/fs"
	log2 "log"
	"sync"
)

var schemas Schema
var groups Groups
var categories Categories
var wappalyzer_fs fs.FS
var icon_url string

// 只需运行一次 - 第一个指纹wr不包含
func InitWappalyzerDB(wr embed.FS, file_ string) error {
	// TODO: 逐行读取，判断是否存在 \\; 如果存在，则判断是否包含version 或 confidence，不存在则打印，根据情况调整正则匹配方式
	wr_sub, err := fs.Sub(wr, "wappalyzer")
	if err != nil {
		return err
	}
	wappalyzer_fs = wr_sub
	schemas = make(Schema)
	technologies := "src/technologies/"
	for i := 0; i < 27; i++ {
		var chr = string(rune(96 + i))
		file_content := make([]byte, 0)
		if chr == "`" {
			file_content = []byte(file_)
		} else {
			file_content, err = fs.ReadFile(wr_sub, technologies+chr+".json")
			if err != nil {
				return err
			}
		}
		var schema Schema
		err = json.Unmarshal(file_content, &schema)
		if err != nil {
			return err
		}
		for k, v := range schema {
			schemas[k] = v
		}
	}

	groups = make(Groups)
	groups_file, err := fs.ReadFile(wr_sub, "src/groups.json")
	if err != nil {
		return err
	}
	err = json.Unmarshal(groups_file, &groups)
	if err != nil {
		return err
	}

	categories = make(Categories)
	categories_file, err := fs.ReadFile(wr_sub, "src/categories.json")
	if err != nil {
		return err
	}
	err = json.Unmarshal(categories_file, &categories)
	if err != nil {
		return err
	}

	icon_null_count := 0
	for name, val := range schemas {
		// 测试是否有未知类型
		_, err = TypeTest(val.Implies)
		if err != nil {
			fmt.Println(name, err)
			return err
		}
		_, err = TypeTest(val.Requires)
		if err != nil {
			fmt.Println(name, err)
			return err
		}
		_, err = TypeTest(val.RequiresCategory)
		if err != nil {
			fmt.Println(name, err)
			return err
		}
		_, err = TypeTest(val.Excludes)
		if err != nil {
			fmt.Println(name, err)
			return err
		}
		_, err = TypeTest(val.DOM)
		if err != nil {
			fmt.Println(name, err)
			return err
		}
		_, err = TypeTest(val.DNS)
		if err != nil {
			fmt.Println(name, err)
			return err
		}
		_, err = TypeTest(val.HTML)
		if err != nil {
			fmt.Println(name, err)
			return err
		}
		_, err = TypeTest(val.TEXT)
		if err != nil {
			fmt.Println(name, err)
			return err
		}
		_, err = TypeTest(val.CSS)
		if err != nil {
			fmt.Println(name, err)
			return err
		}
		_, err = TypeTest(val.Robots)
		if err != nil {
			fmt.Println(name, err)
			return err
		}
		_, err = TypeTest(val.URL)
		if err != nil {
			fmt.Println(name, err)
			return err
		}
		_, err = TypeTest(val.XHR)
		if err != nil {
			fmt.Println(name, err)
			return err
		}
		_, err = TypeTest(val.Meta)
		if err != nil {
			fmt.Println(name, err)
			return err
		}
		_, err = TypeTest(val.ScriptSrc)
		if err != nil {
			fmt.Println(name, err)
			return err
		}

		// 测试ICON是否读取正常
		if val.ICON != "" {
			_, err = fs.ReadFile(wr_sub, "src/drivers/webextension/images/icons/"+val.ICON)
			if err != nil {
				return err
			}
			continue
		}
		icon_null_count++
	}

	log2.Println(fmt.Sprintf("wappalyzer fingers count %d, groups count %d, categories count %d, no icon count %d", len(schemas), len(groups), len(categories), icon_null_count))
	return nil
}

func SetReadICONURL(url string) {
	icon_url = url
}

type Wappalyzer struct {
	Technologies map[string]Technologie
	lock         sync.Mutex
	log          *log.Log
}

type Technologie struct {
	Name       string      `json:"name"`       // 名称
	Confidence int         `json:"confidence"` // 价值
	Version    string      `json:"version"`    // 版本
	Icon       string      `json:"icon"`       // 产品标识
	Website    string      `json:"website"`    // 产品网站
	Cpe        string      `json:"cpe"`        // CPE
	Categories []Categorie `json:"categories"` // 产品分类
}

type Categorie struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func NewWappalyzer(l *log.Log) *Wappalyzer {
	ts := make(map[string]Technologie)
	return &Wappalyzer{Technologies: ts, log: l}
}

func (w *Wappalyzer) GetFingers() map[string]Technologie {
	for name, _ := range w.Technologies {
		if schemas[name].Excludes == nil {
			continue
		}
		switch exc := TypeDetect(schemas[name].Excludes).(type) {
		case string:
			delete(w.Technologies, exc)
			break
		case []string:
			for i := 0; i < len(exc); i++ {
				delete(w.Technologies, exc[i])
			}
			break
		default:
			fmt.Println("found no excludes type", schemas[name].Excludes)
		}
	}
	for name, _ := range w.Technologies {
		if schemas[name].Implies == nil {
			continue
		}
		switch exc := TypeDetect(schemas[name].Implies).(type) {
		case string:
			w.setFinger(name, schemas[name], 100, "")
			break
		case []string:
			for i := 0; i < len(exc); i++ {
				w.setFinger(name, schemas[exc[i]], 100, "")
			}
			break
		default:
			fmt.Println("found no implies type", schemas[name].Excludes)
		}
	}
	for name, value := range w.Technologies {
		if value.Icon == "" {
			continue
		}
		value.Icon = icon_url + value.Icon
		w.Technologies[name] = value
	}
	return w.Technologies
}

func ReadICON(filename string) string {
	icon, err := fs.ReadFile(wappalyzer_fs, "src/drivers/webextension/images/icons/"+filename)
	if err != nil {
		return ""
	}
	return string(icon)
}