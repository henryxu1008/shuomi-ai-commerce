import {expect,it,afterEach} from "vitest";
import {cleanup,render,screen} from "@testing-library/react";
import {ScriptExecutionSummary} from "./script-execution-summary";
afterEach(cleanup);
it("does not present cleanup or an unconfirmed output as successful execution",()=>{
 render(<ScriptExecutionSummary receipt={{outcome:"outcome_unknown",cleanup:"completed",reason:"execution_unconfirmed",inputDigest:"a".repeat(64),outputDigest:"",descriptorDigest:"b".repeat(64)}} before="original" after="unconfirmed"/>);
 expect(screen.getByText(/执行结果未确认/)).toBeInTheDocument();expect(screen.getByText("临时沙箱已清理")).toBeInTheDocument();expect(screen.queryByText(/处理后/)).not.toBeInTheDocument();expect(screen.queryByRole("button")).not.toBeInTheDocument();
});

it("distinguishes an unavailable control connection from confirmed capacity",()=>{
 render(<ScriptExecutionSummary receipt={{outcome:"not_dispatched",cleanup:"not_needed",reason:"control_unavailable",inputDigest:"a".repeat(64),outputDigest:"",descriptorDigest:"b".repeat(64)}} before="Saved title"/>);
 expect(screen.getByText("执行环境连接异常，请联系管理员检查连接、凭据或证书")).toBeInTheDocument();
 expect(screen.queryByText("执行容量暂不可用")).not.toBeInTheDocument();
 expect(screen.getByText("无须清理临时沙箱")).toBeInTheDocument();
 expect(screen.queryByText(/处理后/)).not.toBeInTheDocument();
 expect(screen.queryByRole("button")).not.toBeInTheDocument();
});
