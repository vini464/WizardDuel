package tools

func Enqueue[T any](queue []T, element T) []T {
  queue = append(queue, element)
  return queue
}

func Dequeue[T any](queue []T) (T, []T) {
  element := queue[0]
  if (len(queue) == 1) {
    temp := []T{}
    return element, temp
  }
  return element, queue[1:]
}
