package main

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudwego/eino/compose"
)

// 请假流程状态：同时记录当前步骤
type LeaveState struct {
	Step      string // 当前步骤：askReason / askStartDate / askEndDate / done
	Reason    string
	StartDate string
	EndDate   string
}

// 日期校验
func isValidDate(date string) bool {
	_, err := time.Parse("2006-01-02", date)
	return err == nil
}

// -------------------------- 节点：单步执行 --------------------------

// 节点：询问请假原因（只在 Step=askReason 时执行）
func askReasonStep(ctx context.Context, state *LeaveState) (*LeaveState, error) {
	if state.Step == "askReason" {
		fmt.Println("🤖 请问你请假的原因是什么？（如：生病、事假、年假）")
	}
	return state, nil
}

// 节点：询问开始日期（只在 Step=askStartDate 时执行）
func askStartDateStep(ctx context.Context, state *LeaveState) (*LeaveState, error) {
	if state.Step == "askStartDate" {
		fmt.Println("🤖 请输入请假开始日期（格式：YYYY-MM-DD）")
	}
	return state, nil
}

// 节点：询问结束日期（只在 Step=askEndDate 时执行）
func askEndDateStep(ctx context.Context, state *LeaveState) (*LeaveState, error) {
	if state.Step == "askEndDate" {
		fmt.Println("🤖 请输入请假结束日期（格式：YYYY-MM-DD）")
	}
	return state, nil
}

// 节点：提交请假（只在 Step=done 时执行）
func submitLeaveStep(ctx context.Context, state *LeaveState) (*LeaveState, error) {
	if state.Step == "done" {
		start, _ := time.Parse("2006-01-02", state.StartDate)
		end, _ := time.Parse("2006-01-02", state.EndDate)
		days := int(end.Sub(start).Hours()/24) + 1

		fmt.Println("\n✅ 请假申请提交成功！")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━")
		fmt.Printf("📝 请假原因：%s\n", state.Reason)
		fmt.Printf("📅 开始日期：%s\n", state.StartDate)
		fmt.Printf("📅 结束日期：%s\n", state.EndDate)
		fmt.Printf("📆 请假天数：%d 天\n", days)
		fmt.Println("━━━━━━━━━━━━━━━━━━━━")
	}
	return state, nil
}

// -------------------------- 构建 Chain（单步执行） --------------------------
func buildStepChain() (compose.Runnable[*LeaveState, *LeaveState], error) {
	// 用 Chain 按顺序执行节点，每个节点只在对应 Step 时打印
	chain := compose.NewChain[*LeaveState, *LeaveState]()

	chain.AppendLambda(compose.InvokableLambda(askReasonStep))
	chain.AppendLambda(compose.InvokableLambda(askStartDateStep))
	chain.AppendLambda(compose.InvokableLambda(askEndDateStep))
	chain.AppendLambda(compose.InvokableLambda(submitLeaveStep))

	runnable, err := chain.Compile(context.Background())
	if err != nil {
		return nil, fmt.Errorf("compile chain failed: %w", err)
	}
	return runnable, nil
}

// -------------------------- 主循环：分步交互 --------------------------
func main() {
	fmt.Println("🤖 欢迎使用 Go-Eino 请假系统（输入 exit 退出）")

	// 初始化状态：第一步是 askReason
	state := &LeaveState{
		Step: "askReason",
	}

	chain, err := buildStepChain()
	if err != nil {
		fmt.Println("构建失败：", err)
		return
	}

	ctx := context.Background()

	for {
		// 执行 Chain：只执行当前 Step 对应的打印
		_, err := chain.Invoke(ctx, state)
		if err != nil {
			fmt.Println("执行失败：", err)
			return
		}

		// 如果流程已经完成，退出循环
		if state.Step == "done" {
			break
		}

		// 用户输入
		var input string
		fmt.Print("你：")
		fmt.Scanln(&input)

		if input == "exit" {
			fmt.Println("🤖 再见！")
			return
		}

		// 根据当前 Step 处理输入，并更新状态
		switch state.Step {
		case "askReason":
			state.Reason = input
			state.Step = "askStartDate" // 下一步

		case "askStartDate":
			if !isValidDate(input) {
				fmt.Println("❌ 日期格式错误！请重新输入（YYYY-MM-DD）")
				continue // 不更新 Step，留在当前步骤重试
			}
			state.StartDate = input
			state.Step = "askEndDate" // 下一步

		case "askEndDate":
			if !isValidDate(input) {
				fmt.Println("❌ 日期格式错误！请重新输入（YYYY-MM-DD）")
				continue
			}
			start, _ := time.Parse("2006-01-02", state.StartDate)
			end, _ := time.Parse("2006-01-02", input)
			if end.Before(start) {
				fmt.Println("❌ 结束日期不能早于开始日期！请重新输入")
				continue
			}
			state.EndDate = input
			state.Step = "done" // 进入提交步骤
		}
	}
}
