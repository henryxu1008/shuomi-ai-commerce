// Narrow Next API substitute; HTTP and upgraded WebSocket connections are real.
// It owns no IAM, browser session or business state.
const {createHash}=require('node:crypto');
const {appendFileSync}=require('node:fs');

module.exports=()=>({
 prepare:async()=>{},
 getRequestHandler:()=>((req,res)=>{
  const server=req.socket.server;
  if(!server.listenerCount('upgrade'))server.on('upgrade',(request,socket)=>{
   const accept=createHash('sha1').update(request.headers['sec-websocket-key']+'258EAFA5-E914-47DA-95CA-C5AB0DC85B11').digest('base64');
   socket.write('HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: '+accept+'\r\n\r\n');
   // The fixture sends no frames. A client close can release the socket during
   // failure cleanup without masking the already observed stop result.
   socket.on('data',()=>socket.end());
  });
  res.end('fixture');
 }),
 close:async()=>{
  if(process.env.ISSUE357_NEXT_CLOSE_FILE)appendFileSync(process.env.ISSUE357_NEXT_CLOSE_FILE,'closed\n');
 },
});
