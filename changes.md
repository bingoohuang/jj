# changes

1. 2022-04-03 计算长度
    - `echo '["abc",{"n":1},1,true,null]' | jj -L` => 5
    - `echo  echo '{"n":1,"a":2}' | jj -L` => 3