# changes

1. 2022年04月18日 `JJ_DEPTH=10 jj -R` 指定随机 JSON 生成的元素的个数
2. 2022年04月12日 jj.FreeInnerJSON
3. 2022-04-12日 extends @base64
    - `echo '{"id":"@objectId", "sex":"@random(male,female)", "image":"@base64(file=100.png)"}' | jj -gu > 100.json`
4. 2022-04-03 计算长度
    - `echo '["abc",{"n":1},1,true,null]' | jj #` => 5
    - `echo  echo '{"n":1,"a":2}' | jj #` => 3