# dupimage
Detect duplicated images and gather each group into a unique subfolder.

## usage
```powershell
Usage: dupimage [-adht] ext1 ext2...
  -a    action sort
  -d string
        work directory (default "./")
  -t uint
        duplicate throttle, max is 64 (default 5)
  exts  matching extensions
```
