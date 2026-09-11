"""通过真实伪终端验收密钥遮盖、交互配置及取消行为。"""
import os, pty, select, time, tempfile, json, threading, sys
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
class Handler(BaseHTTPRequestHandler):
 def do_GET(self):
  self.send_response(200);self.send_header('Content-Type','application/json');self.end_headers();self.wfile.write(b'{"user_email":"terminal@example.test"}')
 def log_message(self,*args):pass
server=ThreadingHTTPServer(('127.0.0.1',0),Handler)
thread=threading.Thread(target=server.serve_forever,daemon=True);thread.start()
entry=str(Path(sys.argv[2]).resolve())
node=sys.argv[1]
def run(home,args,steps):
 pid,fd=pty.fork()
 if pid==0:
  env={k:v for k,v in os.environ.items() if not k.startswith('MEMGO_') and not any(x in k for x in ['CLAUDE','CURSOR','CODEX','GEMINI','COPILOT','WINDSURF'])}
  env.update(HOME=home,MEMGO_BASE_URL=f'http://127.0.0.1:{server.server_port}',MEMGO_TELEMETRY='false',NO_COLOR='1')
  os.execvpe(node,[node,entry]+args,env)
 output=b'';index=0;deadline=time.monotonic()+15
 try:
  while time.monotonic()<deadline:
   if select.select([fd],[],[],.1)[0]:
    try:chunk=os.read(fd,8192)
    except OSError:break
    if not chunk:break
    output+=chunk
    if index<len(steps) and steps[index][0].encode() in output:
     os.write(fd,steps[index][1]);index+=1
   done,status=os.waitpid(pid,os.WNOHANG)
   if done:
    assert index==len(steps),(index,output.decode());return os.waitstatus_to_exitcode(status),output.decode()
  done,status=os.waitpid(pid,os.WNOHANG)
  if not done:
   os.kill(pid,9);os.waitpid(pid,0);raise AssertionError('terminal flow timed out: '+output.decode())
  assert index==len(steps),(index,output.decode());return os.waitstatus_to_exitcode(status),output.decode()
 finally:os.close(fd)
try:
 with tempfile.TemporaryDirectory(prefix='memgo-pty-') as home:
  code,output=run(home,['init'],[('Choose',b'2\n'),('API Key',b'terminal-key\n'),('Default User ID',b'alice\n')])
  assert code==0,output
  cfg=Path(home)/'.memgo/config.json';data=json.loads(cfg.read_text());assert data['platform']['api_key']=='terminal-key'
  assert 'terminal-key' not in output,output
  code,output=run(home,['init','--api-key','replacement'],[('Overwrite existing config?',b'n\n')])
  assert code==0,output;assert json.loads(cfg.read_text())['platform']['api_key']=='terminal-key'
  print('PTY: masked key input, default user prompt, connection validation, overwrite cancellation passed')
finally:server.shutdown();server.server_close()
