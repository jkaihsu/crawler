package crawler

import (
  "sync"
)

const MAX_WEB_WORKERS int = 10

/////////////////////////////
// job
/////////////////////////////

type job struct {
  ScrapeQueue  pageStack
  WebWorkers   []*webWorker
  retryLock    sync.Mutex
  Retries      map[*Page]int
  Pages        Pages
  Assets       Assets
  PagesScraped int64
}

func newJob(page *Page) *job {
  j := new(job)
  j.Pages.safeMap.data = make(map[string]interface{})
  j.Assets.safeMap.data = make(map[string]interface{})
  j.Retries = make(map[*Page]int)
  for i := 0; i < MAX_WEB_WORKERS; i++ {
    w := new(webWorker)
    w.job = j
    w.stop = make(chan int)
    j.WebWorkers = append(j.WebWorkers, w)
  }
  j.ScrapeQueue.Push(page)
  return j
}

func (j *job) Start() {
  j.startWorkers()
}

func (j *job) Stop() {
  j.stopWorkers()
}

func (j *job) Done() bool {
  if j.ScrapeQueue.Len() == 0 && j.WorkersDone() {
    return true
  }
  return false
}

func (j *job) WorkersDone() bool {
  for _, w := range j.WebWorkers {
    if w.busy.Value() {
      return false
    }
  }
  return true
}

func (j *job) Requeue(page *Page) {
  j.retryLock.Lock()
  defer j.retryLock.Unlock()
  val := j.Retries[page]
  if val > 2 {
    return
  }
  j.Retries[page] = val + 1
  j.ScrapeQueue.Push(page)
}

func (j *job) startWorkers() {
  for _, w := range j.WebWorkers {
    go w.Scrape()
  }
}

func (j *job) stopWorkers() {
  for _, w := range j.WebWorkers {
    w.stop <- 1
  }
}