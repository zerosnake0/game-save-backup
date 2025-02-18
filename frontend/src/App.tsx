import "./App.css";

import {
  Add,
  AddFiles,
  Backup,
  Backups,
  ChooseDir,
  ChooseFiles,
  Files,
  List,
  Open,
  OpenPath,
  Remove,
  RemoveFile,
  RemoveOne,
  Rename,
  Restore,
  Root,
} from "../wailsjs/go/main/App";
import React, { useEffect, useState } from "react";

interface Context {
  Page: string;
  Current: string;
}

interface ListContainerProps {
  ctx: Context;
  setCtx: (v: Context) => void;
}

const ListContainer = (props: ListContainerProps) => {
  const { ctx, setCtx } = { ...props };

  const [refresh, setRefresh] = useState(true);
  const [list, setList] = useState<Array<string>>([]);
  useEffect(() => {
    List().then((val) => {
      setList(val);
    });
  }, [refresh]);

  const [toAdd, setToAdd] = useState("");
  const [err, setErr] = useState("");

  return (
    <div>
      <div>
        <input onChange={(e) => setToAdd(e.target.value)}></input>
        <button
          onClick={() => {
            setErr("");
            Add(toAdd)
              .then(() => {
                setRefresh(!refresh);
              })
              .catch((e) => {
                setErr(`Add: ${e}`);
              });
          }}
        >
          Add
        </button>
        <button
          onClick={() => {
            Open("").catch((e) => {
              setErr(`Open: ${e}`);
            });
          }}
        >
          Open
        </button>
        <button
          onClick={() => {
            setRefresh(!refresh);
          }}
        >
          Refresh
        </button>
        <div className="err">{err}</div>
      </div>
      <div>
        {list &&
          list.map((v, idx) => {
            return (
              <button
                onClick={() => {
                  //   ctx.Page = "single";
                  //   ctx.Current = v;
                  setCtx({
                    Page: "single",
                    Current: v,
                  });
                }}
                className="square"
                key={idx}
              >
                {v}
              </button>
            );
          })}
      </div>
    </div>
  );
};

interface SingleContainerProps {
  ctx: Context;
  setCtx: (v: Context) => void;
}

const SingleContainer = (props: SingleContainerProps) => {
  const { ctx, setCtx } = { ...props };

  const [removing, setRemoving] = useState(false);
  const [err, setErr] = useState("");
  const [err2, setErr2] = useState("");

  const [files, setFiles] = useState<Array<string>>([]);
  const [refreshFiles, setRefreshFiles] = useState(false);

  useEffect(() => {
    Files(ctx.Current).then((val) => {
      setFiles(val);
    });
  }, [refreshFiles]);

  const [saves, setSaves] = useState<Array<string>>([]);
  const [refreshSaves, setRefreshSaves] = useState(false);

  useEffect(() => {
    Backups(ctx.Current)
      .then((val) => {
        setSaves(val);
      })
      .catch((e) => {
        setErr2(`Backups: ${e}`);
      });
  }, [refreshSaves]);

  const render = () => {
    if (removing) {
      return (
        <React.Fragment>
          <button
            style={{ color: "red" }}
            onClick={() => {
              if (ctx.Current) {
                setErr("");
                Remove(ctx.Current)
                  .then(() => {
                    setCtx({
                      Page: "list",
                      Current: "",
                    });
                  })
                  .catch((e) => {
                    setErr(`Remove: ${e}`);
                  });
              }
            }}
          >
            Remove
          </button>
          <button onClick={() => setRemoving(false)}>Cancel</button>
        </React.Fragment>
      );
    } else {
      return <button onClick={() => setRemoving(true)}>Remove</button>;
    }
  };

  return (
    <div>
      <button
        onClick={() =>
          setCtx({
            Page: "list",
            Current: "",
          })
        }
      >
        Back
      </button>
      <button
        onClick={() => {
          Open(ctx.Current).catch((e) => {
            setErr(`Open ${ctx.Current}: ${e}`);
          });
        }}
      >
        Open
      </button>
      {render()}
      <div className="err">{err}</div>
      <div>current: {ctx.Current}</div>
      {/* 配置 */}
      <div>
        <div>Files:</div>
        {files &&
          files.map((v, idx) => {
            return (
              <div key={idx}>
                {v}{" "}
                <button onClick={() => {
                  OpenPath(v).catch((e) => {
                    setErr2(`Open ${v}: ${e}`);
                  });
                }}>
                  Open
                </button>
                <button
                  onClick={() => {
                    RemoveFile(ctx.Current, v)
                      .then(() => {
                        setRefreshFiles(!refreshFiles);
                      })
                      .catch((e) => {
                        setErr2(`RemoveFile ${v}: ${e}`);
                      });
                  }}
                >
                  x
                </button>
              </div>
            );
          })}
        <button
          onClick={() => {
            ChooseFiles().then((val) => {
              setErr2("");
              AddFiles(ctx.Current, val)
                .then((val) => {
                  setRefreshFiles(!refreshFiles);
                })
                .catch((e) => {
                  setErr2(`AddFiles: ${e}`);
                });
            });
          }}
        >
          AddFile
        </button>
        <button
          onClick={() => {
            ChooseDir().then((val) => {
              setErr2("");
              AddFiles(ctx.Current, [val])
                .then((val) => {
                  setRefreshFiles(!refreshFiles);
                })
                .catch((e) => {
                  setErr2(`AddFiles: ${e}`);
                });
            });
          }}
        >
          AddDir
        </button>
        <div className="err">{err2}</div>
      </div>
      {/* 实际操作 */}
      <div>
        <button
          onClick={() => {
            Backup(ctx.Current)
              .then((val) => {
                setRefreshSaves(!refreshSaves);
              })
              .catch((e) => {
                setErr2(`Backup: ${e}`);
              });
          }}
        >
          Backup
        </button>
        <button onClick={() => { setRefreshSaves(!refreshSaves) }}>
          Refresh
        </button>
        {saves &&
          saves.map((v, idx) => {
            return <SingleSave key={idx}
              name={ctx.Current}
              file={v}
              refreshFunc={() => { setRefreshSaves(!refreshSaves); }}
              errFunc={setErr2}
              canDelete={idx >= 10}
            />
          })}
      </div>
    </div>
  );
};

interface SingleSaveProps {
  name: string;
  file: string;
  refreshFunc: () => void;
  errFunc: (e: any) => void;
  canDelete: boolean;
}

const SingleSave = (props: SingleSaveProps) => {
  const { name, file, refreshFunc, errFunc, canDelete } = { ...props }
  const [newName, setNewName] = useState(file)

  useEffect(() => {
    // console.log(`name changed to ${file}`)
    setNewName(file)
  }, [file])

  return <div >
    {
      (newName == file) && <button className="small"
        onClick={() => {
          Restore(name, file)
            .then(() => {
              refreshFunc()
            })
            .catch((e) => {
              errFunc(`Restore: ${e}`);
            });
        }}
      >
        R
      </button>
    }
    <input className="save" value={newName} defaultValue={file}
      onChange={(e) => {
        setNewName(e.target.value)
      }}
    />
    {
      (newName != file) &&
      <React.Fragment>
        <button className="small"
          onClick={() => {
            Rename(name, file, newName).then(() => {
              refreshFunc()
            }).catch((e) => {
              errFunc(`Rename: ${e}`)
            })
          }}
        >
          Rename
        </button>
        <button className="small"
          onClick={() => {
            setNewName(file)
          }}
        >
          Cancel
        </button>
      </React.Fragment>
    }

    {
      (canDelete) && (newName == file) && <button className="small"
        onClick={() => {
          RemoveOne(name, file)
            .then(() => {
              refreshFunc()
            })
            .catch((e) => {
              errFunc(`RemoveOne: ${e}`);
            });
        }}
      >
        x
      </button>
    }

  </div>
}


function App() {
  const [root, setRoot] = useState("<unknown>");
  const [ctx, setCtx] = useState<Context>({
    Page: "list",
    Current: "",
  });

  useEffect(() => {
    Root().then((val) => {
      setRoot(val);
    });
  }, []);

  const sub = () => {
    switch (ctx.Page) {
      case "single":
        return <SingleContainer ctx={ctx} setCtx={setCtx} />;
      default:
        return <ListContainer ctx={ctx} setCtx={setCtx} />;
    }
  };

  return (
    <div id="App">
      <div id="result" className="result">{`Root: ${root}`}</div>
      {sub()}
    </div>
  );
}

export default App;
