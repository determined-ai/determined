import React from 'react';
import CodeSample from 'hew/CodeSample';


import { Label } from 'hew/Typography';
import { Modal } from 'hew/Modal';


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
            <>
                {fields?.map((lv) => {
                    return (
                        <>
                            <Label>{lv.label}</Label>
                            <CodeSample text={lv.value} />
                        </>
                    )
                })}
            </>

        </Modal>
    );
};
export default TaskConnectModalComponent;
