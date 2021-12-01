import { commandTypeToLabel } from 'constants/states';

import queryString from 'query-string';
import React, { useCallback } from 'react';
import { useParams } from 'react-router-dom';

// import { LogViewerHandles, TAIL_SIZE } from 'components/LogViewer';
import LogViewerCore, { FetchConfig, FetchType } from 'components/LogViewerCore';
import Page from 'components/Page';
// import usePolling from 'hooks/usePolling';
// import useRestApi from 'hooks/useRestApi';
// import { getTaskLogs } from 'services/api';
import { paths } from 'routes/utils';
import { detApi } from 'services/apiConfig';
import { jsonToTaskLog } from 'services/decoder';
// import { TaskLogsParams } from 'services/types';
import { CommandType } from 'types';

import css from './TaskLogs.module.scss';

interface Params {
  taskId: string;
  taskType: string;
}

// interface Queries {
//   id?: string;
// }
type OrderBy = 'ORDER_BY_UNSPECIFIED' | 'ORDER_BY_ASC' | 'ORDER_BY_DESC';

const TaskLogs: React.FC = () => {
  const { taskId, taskType } = useParams<Params>();
  const queries = queryString.parse(location.search);
  const taskTypeLabel = commandTypeToLabel[taskType as CommandType];
  const title = `${queries.id ? `${queries.id} ` : ''}Logs`;
  // const logsRef = useRef<LogViewerHandles>(null);
  // const [ oldestFetchedId, setOldestFetchedId ] = useState(Number.MAX_SAFE_INTEGER);
  // const [ logIdRange, setLogIdRange ] =
  //   useState({ max: Number.MIN_SAFE_INTEGER, min: Number.MAX_SAFE_INTEGER });
  // const baseParams = useMemo(() => ({
  //   tail: TAIL_SIZE,
  //   taskId,
  //   taskType: taskType as CommandType,
  // }), [ taskId, taskType ]);
  // const [ logsResponse, setLogsParams ] =
  //   useRestApi<TaskLogsParams, Log[]>(getTaskLogs, baseParams);
  // const [ pollingLogsResponse, setPollingLogsParams ] =
  //   useRestApi<TaskLogsParams, Log[]>(getTaskLogs, baseParams);

  // const fetchOlderLogs = useCallback((oldestLogId: number) => {
  //   const startLogId = Math.max(0, oldestLogId - TAIL_SIZE);
  //   if (startLogId >= oldestFetchedId) return;
  //   setOldestFetchedId(startLogId);
  //   setLogsParams({ ...baseParams, greaterThanId: startLogId });
  // }, [ baseParams, oldestFetchedId, setLogsParams ]);

  // const fetchNewerLogs = useCallback(() => {
  //   if (logIdRange.max < 0) return;
  //   setPollingLogsParams({ ...baseParams, greaterThanId: logIdRange.max });
  // }, [ baseParams, logIdRange.max, setPollingLogsParams ]);

  // const handleScrollToTop = useCallback((oldestLogId: number) => {
  //   fetchOlderLogs(oldestLogId);
  // }, [ fetchOlderLogs ]);

  // usePolling(fetchNewerLogs);

  // useEffect(() => {
  //   if (!logsResponse.data || logsResponse.data.length === 0) return;

  //   const minLogId = logsResponse.data.first().id;
  //   const maxLogId = logsResponse.data.last().id;
  //   if (minLogId >= logIdRange.min) return;

  //   setLogIdRange({
  //     max: Math.max(logIdRange.max, maxLogId),
  //     min: Math.min(logIdRange.min, minLogId),
  //   });

  //   // If there are new log entries, pass them onto the log viewer.
  //   if (logsRef.current) logsRef.current?.addLogs(logsResponse.data, true);
  // }, [ logIdRange, logsResponse ]);

  // useEffect(() => {
  //   if (!pollingLogsResponse.data || pollingLogsResponse.data.length === 0) return;

  //   const minLogId = pollingLogsResponse.data.first().id;
  //   const maxLogId = pollingLogsResponse.data.last().id;
  //   if (maxLogId <= logIdRange.max) return;

  //   setLogIdRange({
  //     max: Math.max(logIdRange.max, maxLogId),
  //     min: Math.min(logIdRange.min, minLogId),
  //   });

  //   // If there are new log entries, pass them onto the log viewer.
  //   if (logsRef.current) logsRef.current?.addLogs(pollingLogsResponse.data);
  // }, [ logIdRange, pollingLogsResponse.data ]);

  const handleFetch = useCallback((config: FetchConfig, type: FetchType) => {
    const options = {
      follow: false,
      limit: config.limit,
      orderBy: 'ORDER_BY_UNSPECIFIED',
      timestampAfter: '',
      timestampBefore: '',
    };

    if (type === FetchType.Initial) {
      options.orderBy = config.isNewestFirst ? 'ORDER_BY_DESC' : 'ORDER_BY_ASC';
    } else if (type === FetchType.Newer) {
      options.orderBy = 'ORDER_BY_ASC';
      if (config.offsetLog?.time) options.timestampAfter = config.offsetLog.time;
    } else if (type === FetchType.Older) {
      options.orderBy = 'ORDER_BY_DESC';
      if (config.offsetLog?.time) options.timestampBefore = config.offsetLog.time;
    } else if (type === FetchType.Stream) {
      options.follow = true;
      options.limit = 0;
      options.orderBy = 'ORDER_BY_ASC';
      options.timestampAfter = new Date().toISOString();
    }

    return detApi.StreamingJobs.taskLogs(
      taskId,
      options.limit,
      options.follow,
      undefined,
      undefined,
      undefined,
      undefined,
      undefined,
      undefined,
      undefined,
      options.timestampBefore ? new Date(options.timestampBefore) : undefined,
      options.timestampAfter ? new Date(options.timestampAfter) : undefined,
      options.orderBy as OrderBy,
      { signal: config.canceler.signal },
    );
  }, [ taskId ]);

  return (
    // <LogViewer
    //   disableLevel
    //   noWrap
    //   pageProps={{
    //     breadcrumb: [
    //       { breadcrumbName: 'Tasks', path: paths.taskList() },
    //       { breadcrumbName: `${taskTypeLabel} ${taskId.substr(0, 4)}`, path: '#' },
    //     ],
    //     title,
    //   }}
    //   ref={logsRef}
    //   onScrollToTop={handleScrollToTop} />
    <Page
      bodyNoPadding
      breadcrumb={[
        { breadcrumbName: 'Tasks', path: paths.taskList() },
        { breadcrumbName: `${taskTypeLabel} ${taskId.substr(0, 8)}`, path: '#' },
      ]}
      id="task-logs">
      <LogViewerCore
        decoder={jsonToTaskLog}
        title={<div className={css.title}>{title}</div>}
        onFetch={handleFetch}
      />
    </Page>
  );
};

export default TaskLogs;
