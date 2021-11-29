import queryString from 'query-string';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useParams } from 'react-router-dom';

import LogViewer, { LogViewerHandles, TAIL_SIZE } from 'components/LogViewer';
import { commandTypeToLabel } from 'constants/states';
import usePolling from 'hooks/usePolling';
import useRestApi from 'hooks/useRestApi';
import { paths } from 'routes/utils';
import { getTaskLogs } from 'services/api';
import { TaskLogsParams } from 'services/types';
import { CommandType, Log } from 'types';

interface Params {
  taskId: string;
  taskType: string;
}

interface Queries {
  id?: string;
}

const TaskLogs: React.FC = () => {
  const { taskId, taskType } = useParams<Params>();
  const queries: Queries = queryString.parse(location.search);
  const taskTypeLabel = commandTypeToLabel[taskType as CommandType];
  const title = `${taskTypeLabel} Logs${queries.id ? ` (${queries.id})` : ''}`;
  const logsRef = useRef<LogViewerHandles>(null);
  const [ oldestFetchedId, setOldestFetchedId ] = useState(Number.MAX_SAFE_INTEGER);
  const [ logIdRange, setLogIdRange ] =
    useState({ max: Number.MIN_SAFE_INTEGER, min: Number.MAX_SAFE_INTEGER });
  const baseParams = useMemo(() => ({
    tail: TAIL_SIZE,
    taskId,
    taskType: taskType as CommandType,
  }), [ taskId, taskType ]);
  const [ logsResponse, setLogsParams ] =
    useRestApi<TaskLogsParams, Log[]>(getTaskLogs, baseParams);
  const [ pollingLogsResponse, setPollingLogsParams ] =
    useRestApi<TaskLogsParams, Log[]>(getTaskLogs, baseParams);

  const fetchOlderLogs = useCallback((oldestLogId: number) => {
    const startLogId = Math.max(0, oldestLogId - TAIL_SIZE);
    if (startLogId >= oldestFetchedId) return;
    setOldestFetchedId(startLogId);
    setLogsParams({ ...baseParams, greaterThanId: startLogId });
  }, [ baseParams, oldestFetchedId, setLogsParams ]);

  const fetchNewerLogs = useCallback(() => {
    if (logIdRange.max < 0) return;
    setPollingLogsParams({ ...baseParams, greaterThanId: logIdRange.max });
  }, [ baseParams, logIdRange.max, setPollingLogsParams ]);

  const handleScrollToTop = useCallback((oldestLogId: number) => {
    fetchOlderLogs(oldestLogId);
  }, [ fetchOlderLogs ]);

  usePolling(fetchNewerLogs);

  useEffect(() => {
    if (!logsResponse.data || logsResponse.data.length === 0) return;

    const minLogId = logsResponse.data.first().id;
    const maxLogId = logsResponse.data.last().id;
    if (minLogId >= logIdRange.min) return;

    setLogIdRange({
      max: Math.max(logIdRange.max, maxLogId),
      min: Math.min(logIdRange.min, minLogId),
    });

    // If there are new log entries, pass them onto the log viewer.
    if (logsRef.current) logsRef.current?.addLogs(logsResponse.data, true);
  }, [ logIdRange, logsResponse ]);

  useEffect(() => {
    if (!pollingLogsResponse.data || pollingLogsResponse.data.length === 0) return;

    const minLogId = pollingLogsResponse.data.first().id;
    const maxLogId = pollingLogsResponse.data.last().id;
    if (maxLogId <= logIdRange.max) return;

    setLogIdRange({
      max: Math.max(logIdRange.max, maxLogId),
      min: Math.min(logIdRange.min, minLogId),
    });

    // If there are new log entries, pass them onto the log viewer.
    if (logsRef.current) logsRef.current?.addLogs(pollingLogsResponse.data);
  }, [ logIdRange, pollingLogsResponse.data ]);

  return (
    <LogViewer
      disableLevel
      noWrap
      pageProps={{
        breadcrumb: [
          { breadcrumbName: 'Tasks', path: paths.taskList() },
          { breadcrumbName: `${taskTypeLabel} ${taskId.substr(0, 4)}`, path: '#' },
        ],
        title,
      }}
      ref={logsRef}
      onScrollToTop={handleScrollToTop} />
  );
};

export default TaskLogs;
