import os
from textblob import TextBlob
from flask import Flask, request, jsonify

SA_LOGIC_PORT = int(os.getenv('SA_LOGIC_PORT'))

app = Flask(__name__)

@app.route("/healthcheck", methods=['GET'])
def health_check():
    return jsonify(
        result="ok"
    )

@app.route("/analyse/sentiment", methods=['POST'])
def analyse_sentiment():
    sentence = request.get_json()['sentence']
    polarity = TextBlob(sentence).sentences[0].polarity
    return jsonify(
        sentence=sentence,
        polarity=polarity
    )


if __name__ == '__main__':
    app.run(host='0.0.0.0', port=SA_LOGIC_PORT)
