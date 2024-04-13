# 数据库相关

## 库选型
- 常见：
    - [sqlx](https://github.com/jmoiron/sqlx)，随便找的非orm的sql封装库，作为对比参考
    - [Ent](https://entgo.io/docs/getting-started)，来自于[kratos推荐](https://go-kratos.dev/docs/guide/ent)
    - [Gorm](https://gorm.io/docs/)，人尽皆知
    - [Gorm-Gen](https://gorm.io/gen/)，最后采用的

- 示例及其问题：
    - sqlx: 这里的问题是表名user和字段名name都需要以“字面值”的形式写在代码中  
        ```golang
        db.Select(&user, "SELECT * FROM user WHERE name = $1", "pancake")
        ```

    - Ent: 
        - 比起sqlx，ent并不需要“硬编码”，如果 age 列名更改了，重新生成 Ent 代码，则以下代码会报错。
        - Ent的“源”不是数据库，而是 Ent 自己的 schema ，修改数据库结构，是先维护 schema ，再生成数据库操作的代码。所以，即使不需要在“CURD代码”中“硬编码”，但是需要在 schema 中“硬编码”。
        - Ent 特有的 Edges 概念，暂时不是我考虑的。因为我个人支持“不使用外键”。也许以后有机会真的用一次Edges来构造“图”结构，也许真的是“真香”
        ```golang
        // schema
        func (User) Fields() []ent.Field {
            return []ent.Field{
                field.String("name"),
            }
        }
        // update
        user, err = user.Update().    // User update builder.
        SetName("pancake").         // Set a field value.
        Save(ctx)                   // Save and return. 
        ```

    - Gorm: 
        - 和 Ent 类似。接口风格不一样。
        ```golang
        type User struct {
        gorm.Model
        Name  string
        }

        var user User
        db.Model(&user).Update("Name", "pancake")
        db.Model(&user).Updates(User{Name: "pancake"}) 
        ```

    - Gorm-Gen: 
        - user 结构体虽然“存在”，但是将由 gen 工具直接通过实际数据库中的结构来生成代码。不需要我手动维护。我只需要维护好实际数据库即可。
        - curd 代码也不需要“硬编码”
        ```golang
        // select
        user, err := query.User.Where(u.Name.Eq("pancake")).First()
        // update
        u.WithContext(ctx).Where(u.Name.Eq("pancake")).Update(u.Name, "cake")
        ```

## 总结
最终选用 Gorm 搭配 gen 工具。
- 以维护数据库本身为主，以数据库本身为准，来生成代码。数据库改动后，通过代码提示就能找到需要修改的代码。减少数据库结构变更后的工作量。
- 依然关注数据库知识以及 sql 语法，并没有封装成一种“新语言”。这很重要，我不会被 golang 或者某个库绑架，我掌握的是golang和sql两种知识，而不是 golang 和 某个库。我可以用 python 替换 golang，我也可以把 mysql 更换成其他支持 sql 的数据库。