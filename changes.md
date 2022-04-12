# changes

1. 2022-04-12日 extends @base64
    - `echo '{"id":"@objectId", "sex":"@random(male,female)", "image":"@base64(file=100.png)"}' | jj -gu > 100.json`
2. 2022-04-03 计算长度
    - `echo '["abc",{"n":1},1,true,null]' | jj #` => 5
    - `echo  echo '{"n":1,"a":2}' | jj #` => 3