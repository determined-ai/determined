ALTER TABLE allocations
ADD dtrain_port integer not null, 
ADD inter_train_process_comm_port1 integer not null, 
ADD inter_train_process_comm_port2 integer not null, 
ADD c10d_port integer not null; 