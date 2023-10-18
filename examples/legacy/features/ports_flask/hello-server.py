from flask import Flask, jsonify, request

app = Flask(__name__)


@app.route("/hello", methods=["GET"])
def hello():
    if request.method == "GET":
        data = {"data": "Hello World"}
        return jsonify(data)


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=5000, debug=True)
