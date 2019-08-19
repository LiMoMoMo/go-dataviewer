package dataviewer

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"text/template"
	"time"
)

const (
	DEFAULTINTERVAL = 2
	VIEWERPATH      = "/view"
	AJAXPATH        = "/data"
	TYPEPATH        = "/types"
)

type ViewType int32

const (
	TypeValue     ViewType = 0
	TypeBandWidth ViewType = 1
	TypeMemory    ViewType = 2
	TypeRank      ViewType = 3
)

type keyPair struct {
	key   string
	value interface{}
}

type sample struct {
	timeStamp string
	pairs     []keyPair
}

type baseViewer interface {
	GetVal(name string) interface{}
}

type Viewer struct {
	// for http
	addr     string
	interval int
	//
	child baseViewer
	ctx   context.Context
	names map[string]interface{}
	types map[string]ViewType
	//
	eventChan chan Event
}

func (v *Viewer) Register(name string, t ViewType) error {
	if v.names == nil {
		v.names = make(map[string]interface{})
	}
	if v.types == nil {
		v.types = make(map[string]ViewType)
	}
	_, ok := v.names[name]
	if ok {
		return errors.New("This name is exist.")
	}
	v.names[name] = name

	_, ok = v.types[name]
	if ok {
		return errors.New("This name is exist.")
	}
	v.types[name] = t
	return nil
}

func (v *Viewer) SetChild(c baseViewer, ctx context.Context) {
	v.ctx = ctx
	v.child = c
}

func (v *Viewer) SetHttp(addr string, interval int) {
	v.addr = addr

	if interval == 0 {
		v.interval = DEFAULTINTERVAL
	} else {
		v.interval = interval
	}
}

func (v *Viewer) Run() {
	// init eventChan as buffer channel.
	v.eventChan = make(chan Event, 16)
	// run http-server
	go func() {
		http.HandleFunc(VIEWERPATH, v.templeHandle)
		http.HandleFunc(AJAXPATH, v.requestHandle)
		http.HandleFunc(TYPEPATH, v.typeHandle)
		e := http.ListenAndServe(v.addr, nil)
		if e != nil {
			fmt.Println(e.Error())
		}
	}()

	// start eventchannel
	go func() {
		for {
			select {
			case <-v.ctx.Done():
				return
			case e := <-v.eventChan:
				e.setBase(v)
				e.handle()
			}
		}
	}()

	// start read values
	go func() {
		for {
			// read values
			select {
			case <-v.ctx.Done():
				return
			default:
				samp := sample{
					timeStamp: time.Now().Format("2006/01/02 15:04:05"),
					pairs:     make([]keyPair, 0),
				}
				for key, _ := range v.names {
					value := v.child.GetVal(key)
					samp.pairs = append(samp.pairs, keyPair{key, value})
				}
				v.eventChan <- &writeEvent{s: samp}
				// Sleep
				time.Sleep(time.Duration(v.interval) * time.Second)
			}
		}
	}()
}

const tpl = `
<!DOCTYPE html>
<html>

<head>
    <meta charset="utf-8">
    <title>StructViewer</title>
    <script src="https://cdn.bootcss.com/echarts/4.2.1/echarts.common.min.js"></script>
    <script src="http://libs.baidu.com/jquery/2.0.0/jquery.min.js"></script>
</head>

<body>
    <div id="container">
        {{ range . }}
        <div id="{{ . }}" style="width: 100%; height: 25vh;"></div>
        {{ end }}
    </div>
    <script type="text/javascript">
        var BandWidth = {
            formatter: function (value) {
                if ((8 * value / (1024 * 1024 * 1024)) > 1) {
                    return (8 * value / (1024 * 1024 * 1024)).toFixed(2) + " Gbps";
                } else if ((8 * value / (1024 * 1024)) > 1) {
                    return (8 * value / (1024 * 1024)).toFixed(2) + " Mbps";
                } else if ((8 * value / (1024)) > 1) {
                    return (8 * value / (1024)).toFixed(2) + " Kbps";
                } else {
                    return 8 * value + " bps";
                }
            }
        }
        var Memory = {
            formatter: function (value) {
                if ((value / (1024 * 1024 * 1024)) > 1) {
                    return (value / (1024 * 1024 * 1024)).toFixed(2) + " GB";
                } else if ((value / (1024 * 1024)) > 1) {
                    return (value / (1024 * 1024)).toFixed(2) + " MB";
                } else if ((value / (1024)) > 1) {
                    return (value / (1024)).toFixed(2) + " KB";
                } else {
                    return value + " Bytes";
                }
            }
        }
        /////////////////////////////////////////////////////
        var typeMap = {};
        var chartMap = {};
        var datalistMap = {};
        datalistMap["timestamp"] = [];

        function getTypes() {
            $.ajax({
                type: "post",
                async: true,
                url: "types",
                data: {
                    name: "all"
                },
                dataType: "json",
                timeout: 20000,
                success: function (result) {
                    if (result != null) {
                        for (var key in result) {
                            typeMap[key] = result[key]
                        }
                        initOptions()
                    }
                },
                error: function (errorMsg) {
                    console.log(errorMsg)
                }
            })
        }

        function initOptions() {
            var children = document.getElementById('container').childNodes;
            for (var i = 0; i < children.length; i++) {
                if (children[i].nodeName == "#text") {
                    continue;
                }
                var id = children[i].id
                var chart = echarts.init(document.getElementById(id));
                var option;
                if (typeMap[id] != 3) {
                    var axis;
                    if (typeMap[id] === 1) {
                        axis = BandWidth;
                    } else if (typeMap[id] === 2) {
                        axis = Memory;
                    } else if (typeMap[id] === 0) {
                        axis = {};
                    }
                    option = {
                        title: {
                            text: 'Value of:' + id
                        },
                        yAxis: {
                            type: 'value',
                            axisLabel: axis,
                        },
                        xAxis: {
                            type: 'category',
                            data: []
                        },
                        series: [{
                            data: [],
                            type: 'line',
                            smooth: true
                        }]
                    };
                    datalistMap[id] = [];
                } else {
                    option = {
                        title: {
                            text: 'Value of:' + id
                        },
                        xAxis: {
                            type: 'value',
                        },
                        yAxis: {
                            type: 'category',
                            data: []
                        },
                        series: [{
                            data: [],
                            type: 'bar'
                        }]
                    };
                }

                chart.setOption(option);
                chartMap[id] = chart;
            }
            longPolling()
            window.setInterval(longPolling, 2000);
        }

        function longPolling() {
            $.ajax({
                type: "post",
                async: true,
                url: "data",
                data: {
                    name: "all"
                },
                dataType: "json",
                timeout: 20000,
                success: function (result) {
                    if (result != null) {
                        for (var key in result) {
                            if (typeMap[key] != 3) {
                                datalistMap[key].push(result[key][0])
                            } else {
                                datalistMap[key] = result[key][0]
                            }
                        }
                        for (var key in chartMap) {
                            if (typeMap[key] != 3) {
                                var option = {
                                    xAxis: {
                                        type: 'category',
                                        data: datalistMap["timestamp"]
                                    },
                                    series: [{
                                        data: datalistMap[key],
                                        type: 'line',
                                        smooth: true
                                    }]
                                };
                                chartMap[key].setOption(option);
                            } else {
                                var klist = [];
                                var vlist = [];
                                for (var v in datalistMap[key]) {
                                    klist.push(v)
                                    vlist.push(datalistMap[key][v])
                                }
                                var option = {
                                    yAxis: {
                                        type: 'category',
                                        data: klist
                                    },
                                    series: [{
                                        data: vlist,
                                        type: 'bar'
                                    }]
                                };
                                chartMap[key].setOption(option);
                            }
                        }
                    }
                },
                error: function (errorMsg) {
                    console.log(errorMsg)
                }
            })
        }

        getTypes()
    </script>
</body>

</html>
`

func (v *Viewer) templeHandle(w http.ResponseWriter, r *http.Request) {
	// gopath := os.Getenv("GOPATH")
	// filepath := gopath + "/src/github.com/LiMoMoMo/go-dataviewer/src/index.html"
	// filepath = strings.ReplaceAll(filepath, "\\", "/")
	// t, err := template.ParseFiles(filepath)
	//
	t, err := template.New("index").Parse(tpl)
	if err != nil {
		fmt.Println("ParseTemplate Error", err)
		w.Write([]byte("ParseTemplate Error"))
		return
	}
	names := make([]string, 0)
	for key, _ := range v.names {
		names = append(names, key)
	}
	t.Execute(w, names)
}

func (v *Viewer) requestHandle(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		var wg sync.WaitGroup
		event := readEvent{
			callbackWriter: w,
			wg:             &wg,
		}
		wg.Add(1)
		v.eventChan <- &event
		wg.Wait()
	}
}

func (v *Viewer) typeHandle(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		var wg sync.WaitGroup
		event := typeEvent{
			callbackWriter: w,
			wg:             &wg,
		}
		wg.Add(1)
		v.eventChan <- &event
		wg.Wait()
	}
}
