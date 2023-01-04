### timewheel
golang时间轮的实现

### 使用
```go
package main

import (
	"fmt"
	"time"

	"github.com/bosii/timewheel"
)

func main() {

	tw := timewheel.New(1*time.Second, 60)
	// 可调整 默认是 10 和 1024
	tw.SetInitWorkPoolSize(10)
	tw.SetInitWorkerQueueSize(1024)

	tw.Start()

	tw.AddFunc(1*time.Second, func() {
		fmt.Println("hello")
	})

	tw.AddCycleFunc(2*time.Second, func() {
		fmt.Println("world")
	})

	select {}
}

```

### 参考 
参考了这个库 https://github.com/ouqiang/timewheel

