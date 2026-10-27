// Test-only boundary doubles: no Docker, Go, git, sockets or retained runs.
import childProcess from 'node:child_process';
import fs from 'node:fs/promises';
import net from 'node:net';
import {EventEmitter} from 'node:events';
import {PassThrough,Writable} from 'node:stream';
import {syncBuiltinESMExports} from 'node:module';
import {join,resolve,relative} from 'node:path';
import {tmpdir} from 'node:os';

const scenario=process.env.FAILURE_FIXTURE,root=resolve(tmpdir());
const receipt={writes:[]};let protectedDirectory=false;
const write=fs.writeFile,unlink=fs.unlink,lstat=fs.lstat;
const owned=path=>{const r=relative(root,resolve(path));if(r.startsWith('..')||r==='')throw new Error('UNEXPECTED_COMMAND:path');};
process.stdout.write('FIXTURE_LOADED\n');
childProcess.spawn=(command,args)=>{
 const p=new EventEmitter();p.stdout=new PassThrough();p.stderr=new PassThrough();
 const locked=command==='powershell.exe'&&args.some(arg=>arg.includes('ISSUE357_LOCK_FILE'));
 p.stdin=new Writable({write(_chunk,_encoding,done){done()},final(done){done();if(locked)queueMicrotask(()=>p.emit('close',0));}});
 let out='',err='',code=0;
 if(command==='docker') {
  const action=args.slice(2);
  if(action[0]==='version'||(action[0]==='compose'&&action[1]==='version'))out='fixture-version';
  else if(action[1]==='inspect'){code=1;err='No such object';}
  else if(action[0]==='info')out='fixture-version';
  else if(action[0]==='compose'&&action.includes('up')){code=1;err='SYNTHETIC_PRIVATE_STDERR\n';}
  else throw new Error('UNEXPECTED_COMMAND:docker');
 } else if(command==='git') {
  if(args[0]==='rev-parse')out='a'.repeat(40);
  else if(!['status','ls-files'].includes(args[0]))throw new Error('UNEXPECTED_COMMAND:git');
 } else if(command==='go') {
  if(scenario.startsWith('build')){code=1;err='SYNTHETIC_PRIVATE_STDERR\n';}
 } else if(command==='powershell.exe')out=locked?'LOCKED\n':'fixture-sid';
 else if(command==='icacls.exe') {
  owned(args[0]);
  if(!args.includes('/inheritance:r')||!args.includes('*S-1-5-18:(OI)(CI)F'))throw new Error('UNEXPECTED_COMMAND:ACL');
  protectedDirectory=true;
 } else throw new Error('UNEXPECTED_COMMAND:spawn');
 queueMicrotask(()=>{p.stdout.end(out);p.stderr.end(err);if(!locked)p.emit('close',code);});
 return p;
};
let port=12000;
net.createServer=()=>{
 const s=new EventEmitter();let selected;
 s.listen=(preferred,_host,done)=>{
  selected=preferred||++port;
  if(preferred&&scenario==='compose-port-failure') {
   process.stdout.write('FIXTURE_CLEANUP\n');
   queueMicrotask(()=>s.emit('error',Object.assign(new Error('SYNTHETIC_PRIVATE_PORT_FAILURE'),{code:'EADDRINUSE',syscall:'listen',address:'127.0.0.1',port:preferred})));
  } else queueMicrotask(done);
 };
 s.address=()=>({port:selected});s.close=done=>queueMicrotask(done);return s;
};
fs.lstat=async path=>String(path).endsWith(join('node_modules','next','package.json'))?{}:lstat(path);
fs.symlink=async(_target,path)=>{owned(path);await fs.mkdir(path,{recursive:true});};
fs.unlink=async path=>{
 owned(path);
 if(String(path).endsWith(join('ui','node_modules'))) {
  process.stdout.write('FIXTURE_CLEANUP\n');
  if(!scenario.endsWith('cleanup-success'))throw Object.assign(new Error('SYNTHETIC_PRIVATE_UNLINK_FAILURE'),{code:'EPERM',errno:-4048,syscall:'unlink',path});
  return;
 }
 return unlink(path);
};
fs.writeFile=async(path,data,options)=>{
 owned(path);
 if(String(path).endsWith('failure-details.json.tmp')) {
  if(scenario==='evidence-write-failure')throw Object.assign(new Error('SYNTHETIC_PRIVATE_WRITE_FAILURE'),{code:'EACCES'});
  receipt.writes.push({mode:options?.mode,protectedDirectory:process.platform==='win32'?protectedDirectory:true});
  await write(join(root,'protection-receipt.json'),JSON.stringify(receipt));
 }
 return write(path,data,options);
};
syncBuiltinESMExports();
