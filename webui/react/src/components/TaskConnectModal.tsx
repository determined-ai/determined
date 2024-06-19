import React from 'react';
import CodeSample from 'hew/CodeSample';


import { Label } from 'hew/Typography';
import { Modal } from 'hew/Modal';
import css from './TaskConnectModal.module.scss';


export interface Props {
    title?: string;
    fields?: TaskConnectField[];
}

export interface TaskConnectField {
    label: string;
    value: string;
}

const TaskConnectModalComponent: React.FC<Props> = ({
    fields,
    title,
}: Props) => {
    return (
        <Modal
            size="medium"
            title={title ? title : 'Connect to Task'}
        >
            <div className={css.base}>
                {fields?.map((lv) => {
                    return (
                        <div className={css.connectItem}>
                            <Label>{lv.label}</Label>
                            <CodeSample text={lv.value} />
                        </div>
                    )
                })}
            </div>

        </Modal>
    );
};
export default TaskConnectModalComponent;
