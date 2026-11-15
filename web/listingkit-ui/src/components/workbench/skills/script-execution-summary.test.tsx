import {expect,it,afterEach} from "vitest";
import {cleanup,render,screen} from "@testing-library/react";
import {ScriptExecutionSummary} from "./script-execution-summary";
afterEach(cleanup);
it("does not present cleanup or an unconfirmed output as successful execution",()=>{
 render(<ScriptExecutionSummary receipt={{outcome:"outcome_unknown",cleanup:"completed",reason:"execution_unconfirmed",inputDigest:"a".repeat(64),outputDigest:"",descriptorDigest:"b".repeat(64)}} before="original" after="unconfirmed"/>);
 expect(screen.getByText(/执行结果未确认/)).toBeInTheDocument();expect(screen.getByText("临时沙箱已清理")).toBeInTheDocument();expect(screen.queryByText(/处理后/)).not.toBeInTheDocument();expect(screen.queryByRole("button")).not.toBeInTheDocument();
});
