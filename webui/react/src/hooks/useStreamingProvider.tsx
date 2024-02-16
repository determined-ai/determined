import { forEach } from 'lodash';
import React, { createContext, useCallback, useEffect, useMemo } from 'react';

import { serverAddress } from 'routes/utils';
import { Streamable, StreamContent, StreamEntityMap } from 'services/stream';
import { Stream } from 'services/stream/stream';
import projectStore, { mapStreamProject } from 'stores/projects';

type UserStreamingContext = {
  stream?: Stream;
};

export const Streaming = createContext<UserStreamingContext>({
  stream: undefined,
});

export const StreamingProvider: React.FC<React.PropsWithChildren> = ({ children }) => {
  const socketUrl = `${serverAddress().replace('http', 'ws')}/stream`;

  const onUpsert = useCallback((m: Record<string, StreamContent>) => {
    forEach(m, (val, k) => {
      switch (StreamEntityMap[k]) {
        case 'projects':
          projectStore.upsertProject(mapStreamProject(val));
          break;
        default:
          throw new Error(`Unknown stream entity: ${k}`);
      }
    });
  }, []);

  const onDelete = useCallback((entity: Streamable, deleted: Array<number>) => {
    if (deleted.length === 0) return;
    switch (entity) {
      case 'projects':
        for (const d of deleted) {
          projectStore.deleteProject(d);
        }
        break;
      default:
        throw new Error(`Unknown stream entity: ${entity}`);
    }
  }, []);

  const stream = useMemo(
    () => new Stream(socketUrl, onUpsert, onDelete),
    [socketUrl, onUpsert, onDelete],
  );

  useEffect(() => {
    return () => stream.close();
  }, [stream]);

  return (
    <Streaming.Provider
      value={{
        stream,
      }}>
      {children}
    </Streaming.Provider>
  );
};
