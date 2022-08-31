import { Progress, Tooltip, Typography } from 'antd';
import React from 'react';

import Badge, { BadgeType } from 'components/Badge';
import ExperimentActionDropdown from 'components/ExperimentActionDropdown';
import Link from 'components/Link';
import TimeAgo from 'components/TimeAgo';
import TimeDuration from 'components/TimeDuration';
import UserAvatar from 'components/UserAvatar';
import { useStore } from 'contexts/Store';
import { paths } from 'routes/utils';
import Icon from 'shared/components/Icon/Icon';
import { getDuration } from 'shared/utils/datetime';
import { ExperimentItem, Project } from 'types';
import { getProjectExperimentForExperimentItem } from 'utils/experiment';

import css from './ExperimentCard.module.scss';

interface Props {
  experiment: ExperimentItem;
  project?: Project,
}

const ExperimentCard: React.FC<Props> = ({ experiment, project }) => {
  const { auth: { user } } = useStore();

  return (
    <div className={css.base}>
      <div>
        <article className={css.headerContainer}>
          <div className={css.horizotalFlex}>
            <Tooltip title={experiment.resourcePool}>
              <div><Icon name="cluster" /></div>
            </Tooltip>
            {experiment.archived && (
              <Tooltip title="Archive">
                <div><Icon name="archive" /></div>
              </Tooltip>
            )}
          </div>
          <ExperimentActionDropdown
            curUser={user}
            experiment={getProjectExperimentForExperimentItem(experiment, project)}
          />
        </article>
        <article className={css.contentContainer}>
          <Typography.Text strong title={experiment.name}>
            <Link path={paths.experimentDetails(experiment.id)}>
              <span className={css.name}>{experiment.name}</span> ({experiment.id})
            </Link>
          </Typography.Text>
          <Typography.Text className={css.description} ellipsis title={'description'}>
            {experiment.description}
          </Typography.Text>
          <div className={css.metaInfo}>
            <Tooltip title={experiment.startTime?.toLocaleString()}>
              <TimeAgo datetime={experiment.startTime} />
            </Tooltip>
            <TimeDuration duration={getDuration(experiment)} />
            <div className={css.numTrials}>
              {experiment.numTrials} trial {experiment.numTrials > 1 ? 's' : ''}
            </div>
          </div>
        </article>
        <article className={css.footerContainer}>
          <div className={css.horizotalFlex}>
            <Badge state={experiment.state} type={BadgeType.State} />
            <Progress
              percent={((experiment.progress ?? 0) * 100)}
              type="circle"
              width={24}
            />
          </div>
          <div className={css.horizotalFlex}>
            <UserAvatar userId={experiment.userId} />
          </div>
        </article>
      </div>
    </div>
  );
};

export default ExperimentCard;
