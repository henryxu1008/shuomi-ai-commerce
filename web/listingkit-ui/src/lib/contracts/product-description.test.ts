import {expect,it} from "vitest";
import {descriptionPlainText,sanitizeDescriptionHTML,descriptionReferences,descriptionEditorHTML,restoreDescription} from "./product-description";
it("keeps plain content literal, including whitespace and HTML-looking text",()=>{
 expect(descriptionPlainText("<b>literal</b>\n ","")).toBe("<b>literal</b>\n ");
 expect(descriptionEditorHTML("<b>literal</b>\n ")).toContain("&lt;b&gt;literal&lt;/b&gt;");
});
it("projects the same text as the server without image alt or destinations",()=>{
 expect(descriptionPlainText('<h2>Heading</h2><p>One &amp; two<br>line</p><ul><li>First</li><li>Second</li></ul><p><a href="https://example.org">link</a><img src="https://example.org/a.png" alt="ignored"></p>',"html-v1")).toBe("Heading\nOne & two\nline\nFirst\nSecond\nlink");
});
it("filters active content and proxies only allowed pool images",()=>{
 const image="https://images.example.org/a.png";
 const html=sanitizeDescriptionHTML('<script>alert(1)</script><p onclick="x()" style="color:red">safe</p><a href="javascript:alert(1)">x</a><img src="'+image+'" onerror="x()"><img src="https://foreign.example.org/x">',[image],true);
 expect(html).toContain("/api/image-proxy?url=");
 expect(html).not.toMatch(/<script|onclick|style=|javascript:|onerror|foreign/);
 expect(sanitizeDescriptionHTML('<a href="https://example.org">link</a>',[])).toContain('rel="noopener noreferrer nofollow"');
 expect(descriptionReferences('<img src="'+image+'">',"html-v1")).toEqual([image]);
});


it("restores a plain revision without leaving the current rich marker or changing structured fields",()=>{
 const structured={attributes:[],variants:[]};
 const current={title:"new",description:"<p>new</p>",descriptionFormat:"html-v1" as const,structured};
 expect(restoreDescription(current,{title:"old",description:"<b>literal</b>"})).toEqual({title:"old",description:"<b>literal</b>",structured});
 expect(current.descriptionFormat).toBe("html-v1");
});

it("uses the same Unicode whitespace projection as Go",()=>{expect(descriptionPlainText("<p>\u0085x\u0085</p>","html-v1")).toBe("x");expect(descriptionPlainText("<p>\uFEFFx\uFEFF</p>","html-v1")).toBe("\uFEFFx\uFEFF")});
