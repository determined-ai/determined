import { Tooltip } from 'antd';
import React, { useMemo } from 'react';

import { runStateToLabel } from 'constants/states';
import useUI from 'shared/contexts/stores/UI';
import { DarkLight } from 'shared/themes';
import { RunState } from 'types';

import Active from './Active';
import cancle_dark from './cancle-dark.svg';
import cancle_light from './cancle-light.svg';
import complete from './complete.svg';
import error from './error.svg';
import pause_dark from './pause-dark.svg';
import pause_light from './pause-light.svg';
import Queue from './Queue';
import Spinner from './Spinner';

interface Props {
    state: RunState
}

const ExperimentIcons: React.FC<Props> = ({ state }) => {
    const { ui } = useUI();
    const isDark = useMemo (() => ui.darkLight === DarkLight.Dark, [ui]);
    const icon = useMemo(() => {
        switch (state) {
            case RunState.Queued:
                return <Queue />;
            case RunState.Starting:
            case RunState.Pulling:
                return <Spinner type="bowtie" />;
            case RunState.Running:
                return <Spinner type="half" />;
            case RunState.Paused:
                return <img src={isDark ? pause_dark : pause_light} />;
            case RunState.Completed:
                return <img src={complete} />;
            case RunState.Error:
            case RunState.Deleted:
            case RunState.Deleting:
            case RunState.DeleteFailed:
                return <img src={error} />;
            case RunState.Canceled:
            case RunState.StoppingCanceled:
            case RunState.StoppingCompleted:
            case RunState.StoppingError:
            case RunState.StoppingKilled:
                return <img src={isDark ? cancle_dark : cancle_light} />;
            default:
                return <Active />;
        }
    }, [state, isDark]);
    return <Tooltip title={runStateToLabel[state] || '123123'}><span>{icon}</span></Tooltip>;

};

export default ExperimentIcons;
