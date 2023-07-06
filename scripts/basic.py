import http.server
import socketserver
import subprocess

subprocess.run(["python3", "./scripts/number.py"])
class MyHandler(http.server.SimpleHTTPRequestHandler):
    def do_GET(self):
        self.send_response(200)
        self.send_header("Content-type", "text/html")
        self.end_headers()
        self.wfile.write(b"<h1>Hello, World!</h1>")

        # Read numbers from the file
        with open("numbers.txt", "r") as file:
            numbers = file.readlines()

        # HTML to display numbers
        html = "<h1>Numbers 1 to 100:</h1>"
        html += "<ul>"
        for number in numbers:
            html += f"<li>{number.strip()}</li>"
        html += "</ul>"

        # Response
        self.wfile.write(bytes(html, "utf-8"))
        return


PORT = 8000

with socketserver.TCPServer(("", PORT), MyHandler) as httpd:
    print(f"Server started on port {PORT}")
    httpd.serve_forever()
