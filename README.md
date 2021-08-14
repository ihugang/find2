# find2
文件查找统计工具，go编写。

实现功能：
1、统计指定目录下的文件夹和文件数量，以及文件大小总和。
2、搜索指定文件（支持通配符），可以同时将搜索出来的文件copy到一个指定的目录。

Usage: find2 [folder] [-detail] -search:<keywords> -output:<target folder>
          
          -detail                  -> display file
          
          -search: <keywords>      -> search files that name contains keywords
          
          -output: <target folder> -> copy found files to the folder
          
