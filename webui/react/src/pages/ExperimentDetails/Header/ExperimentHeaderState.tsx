import { Button } from 'antd';
import React, { useCallback, useEffect, useState } from 'react';

import Icon from 'components/Icon';
import StopExperimentModal, { ActionType } from 'components/StopExperimentModal';
import { stateToLabel } from 'constants/states';
import { activateExperiment, pauseExperiment } from 'services/api';
import { getStateColorCssVar } from 'themes';
import { ExperimentBase, JobState, RunState } from 'types';

import css from './ExperimentHeaderState.module.scss';

interface Props {
  experiment: ExperimentBase;
}

const PauseButton: React.FC<Props> = ({ experiment }: Props) => {
  const [ isLoading, setIsLoading ] = useState<boolean>(false);

  useEffect(() => {
    setIsLoading(false);
  }, [ experiment.state ]);

  const onClick = useCallback(async () => {
    try {
      setIsLoading(true);
      await pauseExperiment({ experimentId: experiment.id });
    } catch (e) {
      setIsLoading(false);
    }
  }, [ experiment.id ]);

  const classes = [ css.pauseButton ];
  if (isLoading) classes.push(css.loadingButton);

  return (
    <Button
      className={classes.join(' ')}
      ghost={true}
      icon={<Icon name="pause" size="large" />}
      loading={isLoading}
      shape="circle"
      onClick={onClick}
    />
  );
};

const PlayButton: React.FC<Props> = ({ experiment }: Props) => {
  const [ isLoading, setIsLoading ] = useState<boolean>(false);

  useEffect(() => {
    setIsLoading(false);
  }, [ experiment.state ]);

  const onClick = useCallback(async () => {
    try {
      setIsLoading(true);
      await activateExperiment({ experimentId: experiment.id });
    } catch (e) {
      setIsLoading(false);
    }
  }, [ experiment.id ]);

  const classes = [ css.playButton ];
  if (isLoading) classes.push(css.loadingButton);

  return (
    <Button
      className={classes.join(' ')}
      ghost={true}
      icon={<Icon name="play" size="large" />}
      loading={isLoading}
      shape="circle"
      onClick={onClick}
    />
  );
};

const StopButton: React.FC<Props> = ({ experiment }: Props) => {
  const [ isLoading, setIsLoading ] = useState<boolean>(false);
  const [ isOpen, setIsOpen ] = useState<boolean>(false);

  useEffect(() => {
    setIsLoading(false);
  }, [ experiment.state ]);

  const onClose = useCallback((type: ActionType) => {
    setIsLoading(type !== ActionType.None);
    setIsOpen(false);
  }, []);

  const classes = [ css.stopButton ];
  if (isLoading) classes.push(css.loadingButton);

  return (
    <>
      <Button
        className={classes.join(' ')}
        ghost={true}
        icon={<Icon name="stop" size="large" />}
        loading={isLoading}
        shape="circle"
        onClick={() => setIsOpen(true)}
      />
      {isOpen && (
        <StopExperimentModal
          experiment={experiment}
          onClose={onClose}
        />
      )}
    </>
  );
};

const ExperimentHeaderState: React.FC<Props> = ({ experiment }: Props) => {

  const backgroundColor = getStateColorCssVar(experiment.state);
  return (
    <div className={css.base} style={{ backgroundColor }}>
      {experiment.state === RunState.Active && (
        <PauseButton experiment={experiment} />
      )}
      {experiment.state === RunState.Paused && (
        <PlayButton experiment={experiment} />
      )}
      {[ RunState.Active,
        RunState.Paused,
        JobState.QUEUED,
        JobState.SCHEDULED,
        JobState.SCHEDULEDBACKFILLED ].includes(experiment.state) && (
        <StopButton experiment={experiment} />
      )}
      <span className={css.state}>{stateToLabel(experiment.state)}</span>

    </div>
  );
};

export default ExperimentHeaderState;
