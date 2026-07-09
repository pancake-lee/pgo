# 数据库库选型

> 状态：已决策（采用 Gorm + GormGen）

## 候选方案对比

### sqlx

非 ORM 的 SQL 封装库。问题是表名和字段名需要以字面值形式写在代码中：

```go
db.Select(&user, "SELECT * FROM user WHERE name = $1", "pancake")
```

### Ent

Kratos 推荐的方案。不需要在 CURD 代码中硬编码列名，但 Ent 的"源"不是数据库，而是自己的 schema。修改数据库结构需要先维护 schema 再生成代码，本质上还是在 schema 中硬编码。Edges 概念暂时用不上。

```go
// schema 定义
func (User) Fields() []ent.Field {
    return []ent.Field{field.String("name")}
}
// 使用
user, err = user.Update().SetName("pancake").Save(ctx)
```

### Gorm

接口风格不同，但同样需要手动维护 Go 结构体：

```go
type User struct {
    gorm.Model
    Name string
}
db.Model(&user).Update("Name", "pancake")
```

### GormGen（采用）

User 结构体由 gen 工具直接从数据库结构生成，不需要手动维护。CURD 代码也不需要硬编码字符串：

```go
// 查询
user, err := query.User.Where(u.Name.Eq("pancake")).First()
// 更新
u.WithContext(ctx).Where(u.Name.Eq("pancake")).Update(u.Name, "cake")
```

## 决策理由

- **以数据库为准**：维护数据库本身，代码由工具生成。数据库改动后，通过编译错误就能找到需要修改的代码。
- **不封装成"新语言"**：依然关注数据库知识和 SQL 语法，掌握的是可迁移的通用技能，不会被某个库绑定。
