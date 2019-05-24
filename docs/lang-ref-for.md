### For loop

Iterating with values:

```yaml
array:
#@ for i in range(0,3):
- #@ i
- #@ i+1
#@ end
```

Iterating with index:

```yaml
array:
#@ arr = [1,5,{"key":"val"}]
#@ for i in range(len(arr)):
- val: #@ arr[i]
  index: #@ i
#@ end 
```

Use of `continue/break`:

```yaml
array:
#@ for i in range(0,3):
#@   if i == 1:
#@     continue
#@   end
- #@ i
- #@ i+1
#@ end
```
