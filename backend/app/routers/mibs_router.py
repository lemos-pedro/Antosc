from fastapi import APIRouter, Depends, File, UploadFile, HTTPException, Form
from sqlalchemy.orm import Session
from app import database, crud
import os, zipfile, io, datetime, re
from app.config import settings

router = APIRouter(prefix="/api/mibs", tags=["mibs"])
def get_db():
    yield from database.get_db()

@router.post("/upload/")
async def upload_mib(project_id: int = Form(...), file: UploadFile = File(...), db: Session = Depends(get_db)):
    if not file.filename.endswith(".zip"):
        raise HTTPException(status_code=400, detail="Apenas .zip permitido")

    os.makedirs(settings.UPLOAD_DIR, exist_ok=True)
    timestamp = datetime.datetime.utcnow().strftime("%Y%m%d%H%M%S")
    filename = f"{timestamp}_{file.filename}"
    path = os.path.join(settings.UPLOAD_DIR, filename)

    # Save zip
    with open(path, "wb") as f:
        content = await file.read()
        f.write(content)

    # Extract
    extract_dir = path + "_extracted"
    os.makedirs(extract_dir, exist_ok=True)
    with zipfile.ZipFile(path, "r") as zip_ref:
        zip_ref.extractall(extract_dir)

    # Heuristic parse: procurar linhas com ::= (OID definitions)
    mib_files = []
    for root, dirs, files in os.walk(extract_dir):
        for fname in files:
            if fname.lower().endswith(".mib") or fname.lower().endswith(".txt"):
                full = os.path.join(root, fname)
                mib_files.append(full)
                with open(full, "r", encoding="latin-1", errors="ignore") as fh:
                    text = fh.read()
                    # procura algumas ocorrências de OIDs (heurística)
                    oids = re.findall(r"[A-Za-z0-9\-\_]+\s+::=\s+.+", text)
                    oid_sample = oids[0][:200] if oids else None
                    desc = None
                    if oid_sample:
                        desc = oid_sample
                    # salvar no DB por ficheiro
                    crud.create_mib(db, arquivo=fname, oid=oid_sample, descricao=desc, project_id=project_id)

    return {"status": "ok", "files_extracted": len(mib_files)}
