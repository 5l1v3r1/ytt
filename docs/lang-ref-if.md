### If conditional

- if

```yaml
#@ if True:
test1: 123
test2: 124
#@ end
```

- single-node if

```yaml
#@ if/end True:
test1: 123
```

- if-else conditional

```yaml
#@ if True:
test1: 123
#@ else:
test2: 124
#@ end
```

- if-elif-else conditional

```yaml
#@ if True:
test2: 123
#@ elif False:
test2: 124
#@ else:
test2: 125
#@ end
```
