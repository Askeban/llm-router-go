from fastapi import FastAPI
from pydantic import BaseModel
from sentence_transformers import SentenceTransformer
import numpy as np, os
app = FastAPI(title="Prompt Classifier")
class Req(BaseModel): prompt: str
class Resp(BaseModel): category: str; difficulty: str; confidence: float
model = SentenceTransformer(os.environ.get("EMB_MODEL_NAME","sentence-transformers/all-MiniLM-L6-v2"))
REF=[{"label":"coding","text":"Write a function in Python that merges two sorted lists in O(n) time and tests."},{"label":"reasoning","text":"Analyze a scenario with constraints and provide step-by-step logic."},{"label":"math","text":"Solve a math problem with all steps."},{"label":"writing","text":"Rewrite text to improve clarity and tone."},{"label":"support","text":"Troubleshoot a user issue and ask follow-ups."},{"label":"design","text":"Design a scalable backend architecture with components and trade-offs."}]
embs = model.encode([r["text"] for r in REF], convert_to_numpy=True, normalize_embeddings=True); labels=[r["label"] for r in REF]
TECH={"def","class","import","function","var","const","<html>","SELECT","JOIN","O(N)"}; MATH={"proof","integral","equation","theorem"}
def classify_text(prompt:str): v=model.encode([prompt],convert_to_numpy=True,normalize_embeddings=True)[0]; import numpy as np; sims=embs@v; i=int(np.argmax(sims)); cat,conf=labels[i],float(sims[i]); score=(len(prompt)>600)+(len(prompt)>1200)+any(t in prompt for t in TECH)+any(t in prompt.lower() for t in (m.lower() for m in MATH)); diff="hard" if score>=2 else "medium" if score==1 else "easy"; return cat,diff,conf
@app.post("/classify", response_model=Resp)
def classify(req: Req): c,d,cf=classify_text(req.prompt); return Resp(category=c,difficulty=d,confidence=cf)
