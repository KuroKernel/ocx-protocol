"""Tiny Flask-style web app.

Deliberately written with several common security mistakes so the
OCX-signed agent has real bugs to catch in the demo. Do not deploy.
"""
import hashlib
import os
import pickle
import sqlite3
import subprocess

from flask import Flask, redirect, request, send_from_directory

app = Flask(__name__)

# Hard-coded admin token — should be in env/secret manager.
ADMIN_TOKEN = "s3cret-admin-please-rotate"

DB = sqlite3.connect("app.db", check_same_thread=False)


@app.route("/login", methods=["POST"])
def login():
    user = request.form.get("user", "")
    pw = request.form.get("pw", "")
    # SQL injection: user input concatenated into the query.
    cur = DB.execute(
        "SELECT id FROM users WHERE name = '" + user + "' AND pw_md5 = '"
        + hashlib.md5(pw.encode()).hexdigest() + "'"
    )
    row = cur.fetchone()
    if row:
        return {"ok": True, "uid": row[0]}
    return {"ok": False}, 401


@app.route("/files/<path:filename>")
def serve_file(filename):
    # Path traversal: directly forwards user-controlled path.
    return send_from_directory("/var/www/uploads", filename)


@app.route("/redirect")
def open_redirect():
    # Open redirect: trusts whatever URL the caller passes.
    target = request.args.get("next", "/")
    return redirect(target)


@app.route("/ping")
def ping():
    # Command injection: host taken from query and shelled out unquoted.
    host = request.args.get("host", "127.0.0.1")
    out = subprocess.check_output("ping -c 1 " + host, shell=True)
    return out


@app.route("/restore", methods=["POST"])
def restore_session():
    # Insecure deserialization: pickle.loads on a request body.
    blob = request.get_data()
    state = pickle.loads(blob)
    return {"restored": list(state.keys())}


def run() -> None:
    # Debug=True in production exposes the Werkzeug debugger.
    app.run(host="0.0.0.0", port=8080, debug=True)


if __name__ == "__main__":
    run()
