# changes

1. 2022年06月29日 `JJ_N=3 jj -gu name=@姓名 'sex=@random(男,女)' addr=@地址 idcard=@身份证`

    ```sh
    $ JJ_N=2 jj -gu name=@姓名 'sex=@random(男,女)' addr=@地址 idcard=@身份证
    {"addr":"湖北省神农架毾需路3997号洘竐小区7单元1597室","idcard":"374836201410037710","name":"常醦婏","sex":"男"}
    {"addr":"河北省唐山市煦暺路5909号鴸譅小区5单元1254室","idcard":"54619419831203035X","name":"章漀璹","sex":"女"}
    ```

2. 2022年04月18日 `JJ_N=10 jj -R` 指定随机 JSON 生成的元素的个数
3. 2022年04月12日 jj.FreeInnerJSON
4. 2022-04-12日 extends @base64
    - `echo '{"id":"@objectId", "sex":"@random(male,female)", "image":"@base64(file=100.png)"}' | jj -gu > 100.json`
5. 2022-04-03 计算长度
    - `echo '["abc",{"n":1},1,true,null]' | jj #` => 5
    - `echo  echo '{"n":1,"a":2}' | jj #` => 3
