import json
from flask import Flask, request
from langchain.chat_models import ChatOpenAI
from langchain.schema import AIMessage, HumanMessage, SystemMessage
import ai_service.model as model
import qdrant_client

app = Flask(__name__)

client = qdrant_client.QdrantClient(
    url="localhost", prefer_grpc=True
)
# qdrant = Qdrant(
#     client=client, collection_name="my_documents",
#     embedding_function=embeddings.embed_query
# )

@app.route("/")
def root():
    return "Hello GPT!"


chat_llm = ChatOpenAI(model_name="gpt-4", temperature=0)


@app.route("/assistant_query", methods=["POST"])
def assistant_query():
    print(request.json)
    if request.json["messages"] is None:
        return {
            "kind": "chat",
            "content": "Hey, I'm Teleport - a powerful tool that can assist you in managing your Teleport cluster via ChatGPT.",
        }

    messages = model.context(username=request.json["username"])
    for raw_message in request.json["messages"]:
        match raw_message["role"]:
            case "user":
                messages.append(HumanMessage(content=raw_message["content"]))
            case "assistant":
                messages.append(AIMessage(content=raw_message["content"]))
            case "system":
                messages.append(SystemMessage(content=raw_message["content"]))

    model.add_try_extract(messages)
    completion = chat_llm(messages).content
    # check from langchain.output_parsers import StructuredOutputParser, ResponseSchema
    try:
        data = json.loads(completion)
        return {
            "kind": "command",
            "command": data["command"],
            "nodes": data["nodes"],
            "labels": data["labels"],
        }
    except json.JSONDecodeError:
        return {"kind": "chat", "content": completion}
