# 来源信息

- 仓库: Interview
- 文件: docs/Algorithm/DataStructure/Summarization/Binary Search.md
- 许可: CC BY-NC-SA 4.0

---

### Binary Search

```python
def binarySearch(nums, target):
    l, r = 0, len(nums) -1
    while l <= r:
        mid = l + ((r-l) >> 2)
        if nums[mid] > target:
            r = mid - 1
        elif nums[mid] < target:
            l = mid + 1
	else:
	    return mid
    return -1
```
