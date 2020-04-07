import React from 'react';
import styled from 'styled-components';
import { ifProp, theme } from 'styled-tools';
import TimeAgo from 'timeago-react';

import Badge, { BadgeType } from 'components/Badge';
import Icon from 'components/Icon';
import LayoutHelper from 'components/LayoutHelper';
import Link from 'components/Link';
import ProgressBar from 'components/ProgressBar';
import TaskActionDropdown from 'components/TaskActionDropdown';
import { ShirtSize, Theme } from 'themes';
import { RecentTask } from 'types';
import { percent } from 'utils/number';

const TaskCard: React.FC<RecentTask> = (props: RecentTask) => {

  return (
    <Base {...props} crossover data-test="task-card"
      disabled={!props.url} path={props.url ? props.url : '#'}>
      <StyledProgressBar {...props} percent={(props.progress || 0) * 100}
        state={props.state} />
      <LayoutHelper paddingBottom={ShirtSize.medium}
        paddingRight={ShirtSize.big}>
        <LayoutHelper gap="medium">
          <IconBg>
            <Icon name={props.type.toLowerCase()} />
          </IconBg>
          <LayoutHelper column yCenter>
            <TaskName>{props.title}</TaskName>
            <TaskAge>
              <TaskEvent>{props.lastEvent.name}</TaskEvent>
              <TimeAgo datetime={props.lastEvent.date} />
            </TaskAge>
          </LayoutHelper>
        </LayoutHelper>
      </LayoutHelper>
      <LayoutHelper>
        <LayoutHelper fullWidth spaceBetween yCenter>
          <LayoutHelper gap="medium">
            <Badge type={BadgeType.Default}>{props.id.slice(0,4)}</Badge>
            <Badge state={props.state} type={BadgeType.State} />
            {(props.progress !== undefined) && (props.progress !== 1)
                && <Percentage>{percent(props.progress) + '%'}</Percentage>}
          </LayoutHelper>
          <TaskActionDropdown task={props} />
        </LayoutHelper>
      </LayoutHelper>
    </Base>
  );
};

interface RecentTaskWithTheme extends RecentTask {
  theme: Theme;
}

const onHoverCss = (props: RecentTaskWithTheme): string => {
  return props.url ? `&:hover { background-color: ${props.theme.colors.monochrome[13]}; }` : '';
};

const Base = styled(Link)`
  background-color: ${theme('colors.monochrome.14')};
  color: ${theme('colors.monochrome.5')};
  display: block;
  overflow-wrap: break-word;
  padding: ${theme('sizes.layout.big')};
  padding-right: 0;
  position: relative;
  word-break: break-all;
  &:hover { color: ${theme('colors.monochrome.5')}; }
  a { color: ${theme('colors.monochrome.5')}; }
  a:hover { color: ${theme('colors.monochrome.5')}; }
  ${onHoverCss}
`;

const StyledProgressBar = styled(ProgressBar)`
  left: 0;
  position: absolute;
  top: 0;
  visibility: ${ifProp('progress', 'visible', 'hidden')};
  width: 100%;
`;

const IconBg = styled.div`
  align-items: center;
  background-color: white;
  border: ${theme('sizes.border.width')} solid ${theme('colors.monochrome.11')};
  border-radius: ${theme('sizes.border.radius')};
  display: flex;
  flex-shrink: 0;
  height: 4.4rem;
  justify-content: center;
  width: 4.4rem;
`;

const Percentage = styled.div`
  color: black;
  font-size: ${theme('sizes.font.small')};
  font-weight: bold;
  margin-bottom: auto;
  margin-top: auto;
`;

const TaskName = styled.header`
  color: ${theme('colors.monochrome.2')};
  font-size: ${theme('sizes.font.medium')};
  font-weight: bold;
  line-height: ${theme('sizes.font.jumbo')};
  overflow: hidden;
  padding-bottom: ${theme('sizes.layout.tiny')};
  text-overflow: ellipsis;
  white-space: nowrap;
`;

const TaskAge = styled.div`
  font-size: ${theme('sizes.font.small')};
  line-height: ${theme('sizes.font.large')};
`;

const TaskEvent = styled.span`
  padding-right: ${theme('sizes.layout.micro')};
`;

export default TaskCard;
