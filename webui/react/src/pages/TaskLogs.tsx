import { commandTypeToLabel } from 'constants/states';

import queryString from 'query-string';
import React, { useCallback } from 'react';
import { useParams } from 'react-router-dom';

import LogViewerCore, { FetchConfig, FetchType } from 'components/LogViewerCore';
import Page from 'components/Page';
import { paths } from 'routes/utils';
import { detApi } from 'services/apiConfig';
import { jsonToTaskLog } from 'services/decoder';
import { CommandType } from 'types';

import css from './TaskLogs.module.scss';

interface Params {
  taskId: string;
  taskType: string;
}

type OrderBy = 'ORDER_BY_UNSPECIFIED' | 'ORDER_BY_ASC' | 'ORDER_BY_DESC';

const TaskLogs: React.FC = () => {
  const { taskId, taskType } = useParams<Params>();
  const queries = queryString.parse(location.search);
  const taskTypeLabel = commandTypeToLabel[taskType as CommandType];
  const title = `${queries.id ? `${queries.id} ` : ''}Logs`;

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
