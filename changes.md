# changes


1. 2023年12月23日 支持多行/连续JSON取值，`jj -i testdata/line.json name` 会输出每一行的 name 值
2. 2023年05月12日 同步:

   1. [match](https://github.com/tidwall/match) ec90e00 on Apr 1, 2023
   2. [gjson](https://github.com/tidwall/gjson) e14b8d3 on Nov 22, 2022
   3. [sjson](https://github.com/tidwall/sjson) b279807 on Aug 5, 2022
   4. [pjson](https://github.com/tidwall/pjson) 8744e25 on Sep 8, 2022
   5. [tinylru](https://github.com/tidwall/tinylru) 8009823 20 hours ago

3. 2023年02月01日 `jj -gu l1=@ip 'l2=@ip(192.0.2.0/24)' 'l3=@ip(v6)'` 生成随机IP
4. 2022年11月22日 `jj -gu name=@name..jiami` 对生成的随机值加密，解密 `jiami -i 密文 -p 口令`，默认口令 314159，通过环境变量 PASSPHRASE 设置
5. 2022年07月07日 `N=10,5 jj -Ru`  生成10行随机JSON，每行JSON有5个元素
6. 2022年06月29日 `N=3 jj -gu name=@姓名 'sex=@random(男,女)' addr=@地址 idcard=@身份证`

    ```sh
    $ N=2 jj -gu name=@姓名 'sex=@random(男,女)' addr=@地址 idcard=@身份证
    {"addr":"湖北省神农架毾需路3997号洘竐小区7单元1597室","idcard":"374836201410037710","name":"常醦婏","sex":"男"}
    {"addr":"河北省唐山市煦暺路5909号鴸譅小区5单元1254室","idcard":"54619419831203035X","name":"章漀璹","sex":"女"}
    ```

7. 2022年04月18日 `N=10 jj -R` 指定随机 JSON 生成的元素的个数
8. 2022年04月12日 jj.FreeInnerJSON
9. 2022-04-12日 extends @base64
    - `echo '{"id":"@objectId", "sex":"@random(male,female)", "image":"@base64(file=100.png)"}' | jj -gu > 100.json`
10. 2022-04-03 计算长度
     - `echo '["abc",{"n":1},1,true,null]' | jj \#` => 5
     - `echo '{"n":1,"a":2}' | jj \#` => 3
