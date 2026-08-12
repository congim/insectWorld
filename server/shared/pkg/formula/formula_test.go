// Package formula 通用公式引擎单元测试，覆盖ADR-001验收标准：
// 四则运算/运算符优先级/内置函数库/条件if/种子随机确定性/错误用例/验证集8条真实公式。
package formula

import (
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestEngine 创建测试用公式引擎实例。
func newTestEngine() *FormulaEngine {
	return NewFormulaEngine()
}

// mustParse 解析公式，失败即终止测试并输出公式源码。
func mustParse(t *testing.T, engine *FormulaEngine, source string) *Formula {
	t.Helper()
	f, err := engine.Parse(source)
	require.NoError(t, err, "解析公式 %q 应成功", source)
	return f
}

// mustEval 求值公式，失败即终止测试并输出公式源码。
func mustEval(t *testing.T, engine *FormulaEngine, f *Formula, vars map[string]float64, rand RandSource) float64 {
	t.Helper()
	v, err := engine.Eval(f, vars, rand)
	require.NoError(t, err, "求值公式 %q 应成功", f.source)
	return v
}

// TestArithmetic 测试四则运算与括号优先级。
func TestArithmetic(t *testing.T) {
	engine := newTestEngine()
	cases := []struct {
		source string
		want   float64
	}{
		{"1 + 2 * 3", 7},          // 乘法优先级高于加法
		{"(1 + 2) * 3", 9},        // 括号提升优先级
		{"10 / 2 - 1", 4},         // 除法优先于减法
		{"-5 + 10", 5},            // 一元负号
		{"2 * (3 + 4) - 1", 13},   // 括号+混合运算
		{"1 + 2 + 3", 6},          // 加法左结合
		{"8 / 2 / 2", 2},          // 除法左结合
		{"100 * 0.5", 50},         // 小数运算
		{"-1.5 * 2", -3},          // 负数乘法
		{"(1 + 2) * (3 + 4)", 21}, // 多层括号
	}
	for _, tc := range cases {
		t.Run(tc.source, func(t *testing.T) {
			f := mustParse(t, engine, tc.source)
			got := mustEval(t, engine, f, nil, nil)
			assert.InDelta(t, tc.want, got, 1e-9, "公式 %q 结果", tc.source)
		})
	}
}

// TestComparisonAndLogic 测试比较/逻辑运算符返回1.0/0.0及优先级（ADR-001 3.2）。
func TestComparisonAndLogic(t *testing.T) {
	engine := newTestEngine()
	cases := []struct {
		source string
		want   float64
	}{
		{"2 > 1", 1},            // 大于为真
		{"1 > 2", 0},            // 大于为假
		{"2 >= 2", 1},           // 大于等于
		{"3 < 2", 0},            // 小于
		{"2 <= 2", 1},           // 小于等于
		{"1 == 1", 1},           // 相等
		{"1 == 2", 0},           // 不等
		{"1 != 2", 1},           // 不相等为真
		{"1 + 2 > 2 + 0", 1},    // 算术优先级高于比较：3 > 2
		{"1 < 2 and 2 < 3", 1},  // 逻辑与
		{"1 > 2 or 2 < 3", 1},   // 逻辑或
		{"not 0", 1},            // 逻辑非：0为假取反为真
		{"not 5", 0},            // 逻辑非：非0为真取反为假
		{"1 < 2 and 0 or 1", 1}, // and高于or：等价(1<2 and 0) or 1 = 0 or 1
	}
	for _, tc := range cases {
		t.Run(tc.source, func(t *testing.T) {
			f := mustParse(t, engine, tc.source)
			got := mustEval(t, engine, f, nil, nil)
			assert.InDelta(t, tc.want, got, 1e-9, "公式 %q 结果", tc.source)
		})
	}
}

// TestBuiltinFunctions 测试内置函数库：min/max/floor/ceil/round/abs/clamp及函数嵌套。
func TestBuiltinFunctions(t *testing.T) {
	engine := newTestEngine()
	cases := []struct {
		source string
		want   float64
	}{
		{"min(3, 5)", 3},             // 取较小值
		{"max(3, 5)", 5},             // 取较大值
		{"floor(3.7)", 3},            // 向下取整
		{"floor(-1.2)", -2},          // 向下取整（负数向负方向）
		{"ceil(3.2)", 4},             // 向上取整
		{"ceil(-1.2)", -1},           // 向上取整（负数向零方向）
		{"round(2.5)", 3},            // 四舍五入
		{"round(2.4)", 2},            // 四舍五入
		{"abs(-3.5)", 3.5},           // 绝对值
		{"abs(3.5)", 3.5},            // 绝对值（正数不变）
		{"clamp(5, 0, 3)", 3},        // 钳制上界
		{"clamp(-1, 0, 3)", 0},       // 钳制下界
		{"clamp(2, 0, 3)", 2},        // 区间内不变
		{"max(min(3, 5), 4)", 4},     // 函数嵌套
		{"floor(min(3.9, 4.1))", 3},  // 嵌套+取整
		{"min(3, 5) + max(1, 2)", 5}, // 函数与运算符混用
	}
	for _, tc := range cases {
		t.Run(tc.source, func(t *testing.T) {
			f := mustParse(t, engine, tc.source)
			got := mustEval(t, engine, f, nil, nil)
			assert.InDelta(t, tc.want, got, 1e-9, "公式 %q 结果", tc.source)
		})
	}
}

// TestIfConditional 测试条件函数if：真值分支/假值分支/惰性求值/嵌套（ADR-001验证集#6形态）。
func TestIfConditional(t *testing.T) {
	engine := newTestEngine()

	t.Run("首击伤害条件公式", func(t *testing.T) {
		f := mustParse(t, engine, "if(race == 1, dmg * 2.0, dmg)")
		assert.InDelta(t, 240, mustEval(t, engine, f, map[string]float64{"race": 1, "dmg": 120}, nil), 1e-9)
		assert.InDelta(t, 120, mustEval(t, engine, f, map[string]float64{"race": 2, "dmg": 120}, nil), 1e-9)
	})

	t.Run("惰性求值避免除零", func(t *testing.T) {
		// 未选中分支不求值：x=0时10/x不执行，不会触发除数为0错误
		f := mustParse(t, engine, "if(x != 0, 10 / x, 0)")
		assert.InDelta(t, 0, mustEval(t, engine, f, map[string]float64{"x": 0}, nil), 1e-9)
		assert.InDelta(t, 2, mustEval(t, engine, f, map[string]float64{"x": 5}, nil), 1e-9)
	})

	t.Run("条件用逻辑表达式", func(t *testing.T) {
		f := mustParse(t, engine, "if(atk > def and crit == 1, 100, 10)")
		assert.InDelta(t, 100, mustEval(t, engine, f, map[string]float64{"atk": 10, "def": 5, "crit": 1}, nil), 1e-9)
		assert.InDelta(t, 10, mustEval(t, engine, f, map[string]float64{"atk": 10, "def": 5, "crit": 0}, nil), 1e-9)
	})

	t.Run("嵌套if", func(t *testing.T) {
		f := mustParse(t, engine, "if(if(a, 1, 0), 7, 8)")
		assert.InDelta(t, 7, mustEval(t, engine, f, map[string]float64{"a": 1}, nil), 1e-9)
		assert.InDelta(t, 8, mustEval(t, engine, f, map[string]float64{"a": 0}, nil), 1e-9)
	})
}

// TestRandomDeterminism 测试种子随机确定性：同种子同结果、不同种子不同结果（ADR-001验收标准）。
func TestRandomDeterminism(t *testing.T) {
	engine := newTestEngine()
	f := mustParse(t, engine, "atk * random(0.9, 1.1)")

	t.Run("同种子两次求值完全一致", func(t *testing.T) {
		v1 := mustEval(t, engine, f, map[string]float64{"atk": 100}, NewPcg32(42))
		v2 := mustEval(t, engine, f, map[string]float64{"atk": 100}, NewPcg32(42))
		assert.Equal(t, v1, v2, "同种子必须产生完全一致的求值结果")
	})

	t.Run("不同种子结果不同", func(t *testing.T) {
		v1 := mustEval(t, engine, f, map[string]float64{"atk": 100}, NewPcg32(42))
		v3 := mustEval(t, engine, f, map[string]float64{"atk": 100}, NewPcg32(43))
		assert.NotEqual(t, v1, v3, "不同种子应产生不同结果")
	})

	t.Run("无随机公式不依赖随机源", func(t *testing.T) {
		f2 := mustParse(t, engine, "atk * 1.0")
		assert.InDelta(t, 100, mustEval(t, engine, f2, map[string]float64{"atk": 100}, nil), 1e-9)
	})

	t.Run("random三种签名", func(t *testing.T) {
		p := NewPcg32(7)
		f0 := mustParse(t, engine, "random()")
		v0 := mustEval(t, engine, f0, nil, p)
		assert.GreaterOrEqual(t, v0, 0.0)
		assert.Less(t, v0, 1.0, "random()应在[0,1)区间")
		f1 := mustParse(t, engine, "random(10)")
		v1 := mustEval(t, engine, f1, nil, p)
		assert.GreaterOrEqual(t, v1, 0.0)
		assert.Less(t, v1, 10.0, "random(n)应在[0,n)区间")
		f2 := mustParse(t, engine, "random(5, 8)")
		v2 := mustEval(t, engine, f2, nil, p)
		assert.GreaterOrEqual(t, v2, 5.0)
		assert.Less(t, v2, 8.0, "random(a,b)应在[a,b)区间")
	})
}

// TestPcg32FrozenSequence 固化冻结算法回归值：算法任何改动（即使行为等价）都会导致
// 序列漂移破坏跨版本回放一致性，因此用固定种子断言固定输出序列（ADR-001 3.3）。
func TestPcg32FrozenSequence(t *testing.T) {
	t.Run("seed=0", func(t *testing.T) {
		p := NewPcg32(0)
		wantFloats := []float64{0.21723014628514647, 0.36051486316137016, 0.3754446431994438, 0.11023208778351545, 0.8202311447821558}
		wantInts := []int64{7451216774323849838, 7100737942864528205, 6684341139821115963}
		for i, w := range wantFloats {
			assert.InDelta(t, w, p.Float64(), 1e-15, "Float64第%d个输出", i)
		}
		for i, w := range wantInts {
			assert.Equal(t, w, p.Int63(), "Int63第%d个输出", i)
		}
	})

	t.Run("seed=1", func(t *testing.T) {
		p := NewPcg32(1)
		wantFloats := []float64{0.0842725308611989, 0.7505821192171425, 0.6475023522507399, 0.6855632266961038, 0.4223046584520489}
		for i, w := range wantFloats {
			assert.InDelta(t, w, p.Float64(), 1e-15, "Float64第%d个输出", i)
		}
	})

	t.Run("交错消费序列仍确定", func(t *testing.T) {
		// Float64与Int63交错消费时同种子序列一致，覆盖战斗回放按轮次混合消费场景
		p1 := NewPcg32(42)
		p2 := NewPcg32(42)
		for i := 0; i < 4; i++ {
			f1, f2 := p1.Float64(), p2.Float64()
			assert.Equal(t, f1, f2, "第%d轮Float64输出", i)
			i1, i2 := p1.Int63(), p2.Int63()
			assert.Equal(t, i1, i2, "第%d轮Int63输出", i)
		}
	})
}

// TestPcg32Int63n 测试Int63n区间与非法参数panic。
func TestPcg32Int63n(t *testing.T) {
	p := NewPcg32(7)
	for i := 0; i < 1000; i++ {
		v := p.Int63n(100)
		assert.GreaterOrEqual(t, v, int64(0))
		assert.Less(t, v, int64(100))
	}
	assert.Panics(t, func() { p.Int63n(0) }, "n<=0应按接口契约panic")
	assert.Panics(t, func() { p.Int63n(-1) }, "n<=0应按接口契约panic")
}

// TestVerificationSet 验证集8条真实昆虫公式（ADR-001 1.3节）全部可编译可求值，
// 结果与手工计算一致；#1含random验证区间与确定性。
func TestVerificationSet(t *testing.T) {
	engine := newTestEngine()
	seed := uint64(20260811) // 固定种子，保证#1随机公式断言稳定
	cases := []struct {
		name      string
		source    string
		vars      map[string]float64
		want      float64
		hasRandom bool
	}{
		{
			name:      "#1伤害=攻击×(1-防御/100)×克制系数×random(0.9,1.1)",
			source:    "atk * (1 - def/100) * counter_mod * random(0.9, 1.1)",
			vars:      map[string]float64{"atk": 100, "def": 20, "counter_mod": 1.5},
			hasRandom: true,
		},
		{
			name:   "#2血量=类型基础×(1+等级×0.1)×(1+种族防御修正)",
			source: "base_hp * (1 + level * 0.1) * (1 + race_def_mod)",
			vars:   map[string]float64{"base_hp": 100, "level": 3, "race_def_mod": 0.2},
			want:   156,
		},
		{
			name:   "#3攻击=类型基础×(1+等级×0.1)×(1+种族攻击修正)",
			source: "base_atk * (1 + level * 0.1) * (1 + race_atk_mod)",
			vars:   map[string]float64{"base_atk": 50, "level": 3, "race_atk_mod": 0.1},
			want:   71.5,
		},
		{
			name:   "#4升级成本=基础成本×(1+等级×0.5)",
			source: "base_cost * (1 + level * 0.5)",
			vars:   map[string]float64{"base_cost": 1000, "level": 4},
			want:   3000,
		},
		{
			name:   "#5战斗力=Σ(单位血量×攻击×克制系数)/100",
			source: "(unit_hp * atk * counter_mod) / 100",
			vars:   map[string]float64{"unit_hp": 100, "atk": 50, "counter_mod": 1.5},
			want:   75,
		},
		{
			name:   "#6首击伤害=if(种族==螳螂, 伤害×2.0, 伤害)",
			source: "if(race == 1, dmg * 2.0, dmg)",
			vars:   map[string]float64{"race": 1, "dmg": 120},
			want:   240,
		},
		{
			name:   "#7保留量=min(原值×keep_ratio, keep_amount)",
			source: "min(value * keep_ratio, keep_amount)",
			vars:   map[string]float64{"value": 1000, "keep_ratio": 0.5, "keep_amount": 300},
			want:   300,
		},
		{
			name:   "#8基础伤害=max(atk×counter-def×terrain_mod, 1)",
			source: "max(atk * counter - def * terrain_mod, 1)",
			vars:   map[string]float64{"atk": 100, "counter": 1.5, "def": 80, "terrain_mod": 1.2},
			want:   54,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f, err := engine.Parse(tc.source)
			require.NoError(t, err, "验证集公式 %q 必须可编译（ADR-001 1.3）", tc.source)
			got, err := engine.Eval(f, tc.vars, NewPcg32(seed))
			require.NoError(t, err, "验证集公式 %q 必须可求值（ADR-001 1.3）", tc.source)
			if tc.hasRandom {
				// #1含random：基础值120×[0.9,1.1)∈[108,132)，并验证确定性
				assert.GreaterOrEqual(t, got, 108.0, "随机伤害应不低于区间下界")
				assert.Less(t, got, 132.0, "随机伤害应低于区间上界")
				got2, err := engine.Eval(f, tc.vars, NewPcg32(seed))
				require.NoError(t, err)
				assert.Equal(t, got, got2, "同种子重放必须完全一致（ADR-001验收标准）")
				return
			}
			assert.InDelta(t, tc.want, got, 1e-9, "公式 %q 结果与手工计算一致", tc.source)
		})
	}
}

// TestRegisterAndEvalByID 测试公式注册与按ID求值，含重复注册/空ID/nil公式等错误用例。
func TestRegisterAndEvalByID(t *testing.T) {
	engine := newTestEngine()
	f := mustParse(t, engine, "atk * 2")

	t.Run("正常注册", func(t *testing.T) {
		err := engine.Register("basic_damage", f)
		assert.NoError(t, err)
	})
	t.Run("重复注册", func(t *testing.T) {
		err := engine.Register("basic_damage", f)
		assert.Error(t, err, "重复注册应报错，禁止覆盖已注册公式")
	})
	t.Run("空ID注册", func(t *testing.T) {
		err := engine.Register("", f)
		assert.Error(t, err)
	})
	t.Run("nil公式注册", func(t *testing.T) {
		err := engine.Register("nil_formula", nil)
		assert.Error(t, err)
	})
	t.Run("EvalByID已注册公式", func(t *testing.T) {
		got, err := engine.EvalByID("basic_damage", map[string]float64{"atk": 50}, nil)
		assert.NoError(t, err)
		assert.InDelta(t, 100, got, 1e-9)
	})
	t.Run("EvalByID未注册公式", func(t *testing.T) {
		_, err := engine.EvalByID("not_registered", nil, nil)
		assert.Error(t, err)
	})
	t.Run("Eval nil公式", func(t *testing.T) {
		_, err := engine.Eval(nil, nil, nil)
		assert.Error(t, err)
	})
}

// TestParseErrors 测试解析期错误用例：语法错误/未知函数/参数个数/非法字符（ADR-001 3.4 Validate）。
func TestParseErrors(t *testing.T) {
	engine := newTestEngine()
	cases := []struct {
		name   string
		source string
	}{
		{"表达式不完整", "1 +"},
		{"缺少右括号", "(1 + 2"},
		{"多余token", "1 2"},
		{"未知函数", "unknown_func(1, 2)"},
		{"函数参数过少", "min(1)"},
		{"函数参数过多", "min(1, 2, 3)"},
		{"floor缺参数", "floor()"},
		{"floor参数过多", "floor(1, 2)"},
		{"random参数过多", "random(1, 2, 3)"},
		{"if参数过少", "if(1, 2)"},
		{"if参数过多", "if(1, 2, 3, 4)"},
		{"非法字符", "1 # 2"},
		{"数字格式错误", "1..2"},
		{"空公式", ""},
		{"仅括号", "()"},
		{"函数首个参数缺失", "min(, 2)"},
		{"尾随逗号", "random(1,)"},
		{"单个等号不支持", "a = 1"},
		{"逻辑与作为变量", "a + and"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := engine.Parse(tc.source)
			assert.Error(t, err, "公式 %q 应解析失败", tc.source)
		})
	}

	t.Run("未知函数错误含修复提示", func(t *testing.T) {
		_, err := engine.Parse("foo(1)")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "未定义", "错误应说明函数未定义")
	})
}

// TestNestedDepthLimit 测试表达式嵌套深度限制：≤maxDefaultDepth(10)可编译，超限拒绝编译。
func TestNestedDepthLimit(t *testing.T) {
	engine := newTestEngine()

	t.Run("10层括号在限制内", func(t *testing.T) {
		source := strings.Repeat("(", 10) + "1" + strings.Repeat(")", 10)
		f := mustParse(t, engine, source)
		assert.InDelta(t, 1, mustEval(t, engine, f, nil, nil), 1e-9)
	})
	t.Run("11层括号超限", func(t *testing.T) {
		source := strings.Repeat("(", 11) + "1" + strings.Repeat(")", 11)
		_, err := engine.Parse(source)
		assert.Error(t, err, "11层括号应超限拒绝编译")
	})
	t.Run("10层函数嵌套在限制内", func(t *testing.T) {
		source := buildNestedMin(10)
		f := mustParse(t, engine, source)
		assert.InDelta(t, 1, mustEval(t, engine, f, nil, nil), 1e-9)
	})
	t.Run("11层函数嵌套超限", func(t *testing.T) {
		source := buildNestedMin(11)
		_, err := engine.Parse(source)
		assert.Error(t, err, "11层函数嵌套应超限拒绝编译")
	})
	t.Run("10层一元负号在限制内", func(t *testing.T) {
		source := strings.Repeat("-", 10) + "1"
		f := mustParse(t, engine, source)
		assert.InDelta(t, 1, mustEval(t, engine, f, nil, nil), 1e-9)
	})
	t.Run("11层一元负号超限", func(t *testing.T) {
		source := strings.Repeat("-", 11) + "1"
		_, err := engine.Parse(source)
		assert.Error(t, err, "11层一元负号应超限拒绝编译")
	})
}

// buildNestedMin 构造n层嵌套的min(1, min(1, ...))表达式，用于深度限制测试。
func buildNestedMin(n int) string {
	return strings.Repeat("min(1, ", n) + "1" + strings.Repeat(")", n)
}

// TestMissingVariableAndNilRand 测试缺失变量按0处理、nil随机源时random返回0（ADR-001 3.6语义）。
func TestMissingVariableAndNilRand(t *testing.T) {
	engine := newTestEngine()

	t.Run("缺失变量按0处理不报错", func(t *testing.T) {
		f := mustParse(t, engine, "atk + def")
		assert.InDelta(t, 5, mustEval(t, engine, f, map[string]float64{"atk": 5}, nil), 1e-9)
	})

	t.Run("nil随机源random返回0", func(t *testing.T) {
		f := mustParse(t, engine, "random(10)")
		assert.InDelta(t, 0, mustEval(t, engine, f, nil, nil), 1e-9)
	})
}

// TestEvalErrors 测试求值期错误：除数为0返回错误（配置防御，避免Inf漂移到int64转换）。
func TestEvalErrors(t *testing.T) {
	engine := newTestEngine()
	cases := []struct {
		name   string
		source string
		vars   map[string]float64
	}{
		{"除数为0字面量", "10 / 0", nil},
		{"除数为0表达式", "10 / (x - x)", map[string]float64{"x": 5}},
		{"选中分支除零", "if(1, 1 / 0, 2)", nil}, // 条件为真时选中分支求值应报错
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := mustParse(t, engine, tc.source)
			_, err := engine.Eval(f, tc.vars, nil)
			assert.Error(t, err, "公式 %q 求值应失败", tc.source)
		})
	}
}

// TestConcurrentEval 测试同一公式并发求值：只读Formula共享安全，同种子结果一致（ADR-001 3.6并发求值）。
func TestConcurrentEval(t *testing.T) {
	engine := newTestEngine()
	f := mustParse(t, engine, "atk * (1 + level * 0.1) * random(0.9, 1.1)")
	vars := map[string]float64{"atk": 100, "level": 3}
	const goroutines = 16
	results := make([]float64, goroutines)
	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			v, err := engine.Eval(f, vars, NewPcg32(20260811))
			if err != nil {
				t.Errorf("并发求值失败: %v", err)
				return
			}
			results[idx] = v
		}(i)
	}
	wg.Wait()
	for i := 1; i < goroutines; i++ {
		assert.Equal(t, results[0], results[i], "同一公式+变量+种子并发求值必须一致")
	}
}
