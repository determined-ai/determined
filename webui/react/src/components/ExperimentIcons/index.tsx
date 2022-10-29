import { Tooltip } from 'antd';
import { runStateToLabel } from 'constants/states';
import React, {useMemo} from 'react';
import { RunState } from 'types';
import Active from './Active';
import Queue from './Queue';
import Spinner from './Spinner';

interface Props {
    state: RunState
}

const ExperimentIcons: React.FC<Props> = ({state}) => {
    const icon = useMemo(() => {
        switch(state) {
            case RunState.Queued:
                return <Queue />

            case RunState.Starting:
                return <Spinner type='bowtie' />

            case RunState.Running:
                return <Spinner type='half' />
            default:
                return <Active />
        }
    }, [state])

    return <Tooltip title={runStateToLabel[state]}>{icon}</Tooltip>
   
}

export default ExperimentIcons
