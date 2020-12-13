# timestamps
Run a program with timestamped output

## synopsis

```
ts - run a command with timestamped output

usage:
  ts [ options ] cmd args...

options:
  -format string
    	timestamp format (default "default")
  -millis
    	calculate timestamps in milliseconds since program start.
  -tabs
    	use tabs rather than spaces after the timestamp
  -utc
    	use utc timestamps instead of localtime ones.
  -verbose
    	verbose output
```

## installation

$ make && sudo make install

## build dependencies

golang >= 1.13
