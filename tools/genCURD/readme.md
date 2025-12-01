# genCURD

这是一个**基于Gorm生成的Model**生成**CURD服务接口代码**的生成工具

- 人工编写，abandon_code对应的sql/proto以及service代码，这是我们的模板。
- abandon命名只是我为了排序第一的灵感单词，刚好也能表达“这不是一个真正的功能模块”。
- abandon_code的代码里包含了类似 `// MARK REPLACE XXX START/END`这样的标识
- 根据abandon_code及其 `MARK`标识，通过 `genCURD`可以生成出其他表的CURD服务接口代码

相比之下，更加常见的是利用 `text/template`库，在代码中利用 `{{.FieldName}}`表示替换位置。
这种方案中，带有 `{{.FieldName}}`的代码无法被VSCode做代码静态分析，也无法被正常编译。
所以除非模板代码已经非常稳定，否则维护这样的模板代码并不容易。

与其他生成工具不同的是，abandon_code相关代码是一份**正常**的代码。
而abandon_code的维护和正常开发几乎没有区别，只需要提供几个 `MARK`标识即可。
