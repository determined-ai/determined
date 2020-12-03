import { Button } from 'antd';
import moment, { Moment } from 'moment';
import React, { useEffect, useMemo, useState } from 'react';

import { V1TrialLogsFieldsResponse } from '../services/api-ts-sdk';
import { detApi } from '../services/apiConfig';
import { consumeStream } from '../services/utils';

import DatePickerFilter from './DatePickerFilter';
import MultiSelect from './MultiSelect';
import ResponsiveFilters from './ResponsiveFilters';
import css from './TrialLogFilters.module.scss';

export enum LogLevelFromApi {
  Unspecified = 'LOG_LEVEL_UNSPECIFIED',
  Trace = 'LOG_LEVEL_TRACE',
  Debug = 'LOG_LEVEL_DEBUG',
  Info = 'LOG_LEVEL_INFO',
  Warning = 'LOG_LEVEL_WARNING',
  Error = 'LOG_LEVEL_ERROR',
  Critical = 'LOG_LEVEL_CRITICAL',
}

export interface TrialLogFiltersInterface {
  agentIds?: Array<string>,
  containerIds?: Array<string>,
  rankIds?: Array<number>,
  levels?: Array<LogLevelFromApi>,
  stdtypes?: Array<string>,
  sources?: Array<string>,
  timestampBefore?: Moment,
  timestampAfter?: Moment,
}

interface Props {
  filter: TrialLogFiltersInterface;
  trialId: number;
  onChange?: (newFilters: TrialLogFiltersInterface) => void;
}

const TrialLogFilters: React.FC<Props> = ({ filter, onChange, trialId }: Props) => {
  const [ availableFilters, setAvailableFilters ] = useState<V1TrialLogsFieldsResponse|null>(null);

  const broadcastChange = (newFilter: TrialLogFiltersInterface) => {
    if (typeof onChange === 'function') {
      onChange(newFilter);
    }
  };

  const onAgentChange = (value: (number|string)[]) => broadcastChange({
    ...filter,
    agentIds: value.map((item) => String(item)),
  });

  const onClear = () => broadcastChange({});

  const onContainerChange = (value: (number|string)[]) => broadcastChange({
    ...filter,
    containerIds: value.map((item) => String(item)),
  });

  const onRankChange = (value: (number|string)[]) => broadcastChange({
    ...filter,
    rankIds: value.map((item) => Number(item)),
  });

  const onLevelChange = (value: (number|string)[]) => broadcastChange({
    ...filter,
    levels: value.map((item) => String(item) as LogLevelFromApi),
  });

  const onDateChange = (key: string, date: Moment|null) => {
    if (!date) {
      return;
    }

    // receiving a moment with user timezone. need to keep the selected date/time but
    // set the timezone to UTC.
    const iso8601StringNoTz = date.format().substr(0, 19);
    const momentUtc = moment.utc(iso8601StringNoTz);
    broadcastChange({
      ...filter,
      [key]: momentUtc,
    });
  };

  const onAfterDateChange = (date: Moment|null) => onDateChange('timestampAfter', date);

  const onBeforeDateChange = (date: Moment|null) => onDateChange('timestampBefore', date);

  const logLevelList = useMemo(() => {
    return Object.entries(LogLevelFromApi)
      .filter(([ key ]) => key !== 'Unspecified')
      .map(([ key, value ]) => ({ label: key, value }));
  }, []);

  useEffect(() => {
    consumeStream<V1TrialLogsFieldsResponse>(
      detApi.StreamingExperiments.determinedTrialLogsFields(
        trialId,
        true,
      ),
      event => setAvailableFilters(event),
    );
  }, [ trialId ]);

  return (
    <ResponsiveFilters>
      <MultiSelect
        label="Agents"
        options={availableFilters?.agentIds || []}
        value={filter.agentIds || []}
        onChange={onAgentChange}
      />
      <MultiSelect
        label="Containers"
        options={availableFilters?.containerIds || []}
        value={filter.containerIds || []}
        onChange={onContainerChange}
      />
      <MultiSelect
        label="Ranks"
        options={availableFilters?.rankIds || []}
        value={filter.rankIds || []}
        onChange={onRankChange}
      />
      <MultiSelect
        label="Level"
        options={logLevelList}
        value={filter.levels || []}
        onChange={onLevelChange}
      />
      <DatePickerFilter
        label="After"
        value={filter.timestampAfter}
        onChange={onAfterDateChange}
      />
      <DatePickerFilter
        label="Before"
        value={filter.timestampBefore}
        onChange={onBeforeDateChange}
      />
      <Button
        className={css.clearButton}
        onClick={onClear}
      >
        Clear
      </Button>
    </ResponsiveFilters>
  );
};

export default TrialLogFilters;
