import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { useParams, useSearchParams } from 'react-router-dom';

import LogViewer, {
  FetchConfig,
  FetchDirection,
  FetchType,
} from 'components/kit/LogViewer/LogViewer';
import LogViewerSelect, { Filters } from 'components/kit/LogViewer/LogViewerSelect';
import { Settings, settingsConfigForTask } from 'components/kit/LogViewer/LogViewerSelect.settings';
import Page from 'components/Page';
import { commandTypeToLabel } from 'constants/states';
import { useSettings } from 'hooks/useSettings';
import { paths } from 'routes/utils';
import { detApi } from 'services/apiConfig';
import { mapV1LogsResponse } from 'services/decoder';
import { readStream } from 'services/utils';
import { CommandType } from 'types';

import css from './TaskLogs.module.scss';

type Params = {
  taskId: string;
  taskType: string;
};

interface Props {
  headerComponent?: React.ReactNode;
  onCloseLogs?: () => void;
  taskId: string;
  taskType: string;
}
type OrderBy = 'ORDER_BY_UNSPECIFIED' | 'ORDER_BY_ASC' | 'ORDER_BY_DESC';

export const TaskLogsWrapper: React.FC = () => {
  const { taskId, taskType } = useParams<Params>();
  return <TaskLogs taskId={taskId ?? ''} taskType={taskType ?? ''} />;
};
const TaskLogs: React.FC<Props> = ({ taskId, taskType, onCloseLogs, headerComponent }: Props) => {
  const [filterOptions, setFilterOptions] = useState<Filters>({});
  const [searchParams] = useSearchParams();

  const taskTypeLabel = commandTypeToLabel[taskType as CommandType];
  const title = `${searchParams.has('id') ? `${searchParams.get('id')} ` : ''}Logs`;

  const taskSettingsConfig = useMemo(() => settingsConfigForTask(taskId), [taskId]);
  const { resetSettings, settings, updateSettings } = useSettings<Settings>(taskSettingsConfig);

  const filterValues: Filters = useMemo(
    () => ({
      agentIds: settings.agentId,
      containerIds: settings.containerId,
      levels: settings.level,
      rankIds: settings.rankId,
      searchText: settings.searchText,
    }),
    [settings],
  );

  const handleFilterChange = useCallback(
    (filters: Filters) => {
      updateSettings({
        agentId: filters.agentIds,
        allocationId: filters.allocationIds,
        containerId: filters.containerIds,
        level: filters.levels,
        rankId: filters.rankIds,
        searchText: filters.searchText,
      });
    },
    [updateSettings],
  );

  const handleFilterReset = useCallback(() => resetSettings(), [resetSettings]);

  const handleFetch = useCallback(
    (config: FetchConfig, type: FetchType) => {
      const options = {
        follow: false,
        limit: config.limit,
        orderBy: 'ORDER_BY_UNSPECIFIED',
        timestampAfter: '',
        timestampBefore: '',
      };

      if (type === FetchType.Initial) {
        options.orderBy =
          config.fetchDirection === FetchDirection.Older ? 'ORDER_BY_DESC' : 'ORDER_BY_ASC';
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
        settings.allocationId,
        settings.agentId,
        settings.containerId,
        settings.rankId,
        settings.level,
        undefined,
        undefined,
        options.timestampBefore ? new Date(options.timestampBefore) : undefined,
        options.timestampAfter ? new Date(options.timestampAfter) : undefined,
        options.orderBy as OrderBy,
        settings.searchText,
        { signal: config.canceler.signal },
      );
    },
    [settings, taskId],
  );

  useEffect(() => {
    const canceler = new AbortController();

    readStream(
      detApi.StreamingJobs.taskLogsFields(taskId, true, { signal: canceler.signal }),
      (event) => setFilterOptions(event as Filters),
    );

    return () => canceler.abort();
  }, [taskId]);

  const logFilters = (
    <div className={css.filters}>
      <LogViewerSelect
        options={filterOptions}
        showSearch={true}
        values={filterValues}
        onChange={handleFilterChange}
        onReset={handleFilterReset}
      />
    </div>
  );

  return (
    <Page
      bodyNoPadding
      breadcrumb={[
        { breadcrumbName: 'Tasks', path: paths.taskList() },
        { breadcrumbName: `${taskTypeLabel} ${taskId.substring(0, 8)}`, path: '#' },
      ]}
      headerComponent={headerComponent}
      id="task-logs"
      title={title}>
      <LogViewer
        decoder={mapV1LogsResponse}
        handleCloseLogs={onCloseLogs}
        title={logFilters}
        onFetch={handleFetch}
      />
    </Page>
  );
};

export default TaskLogs;
