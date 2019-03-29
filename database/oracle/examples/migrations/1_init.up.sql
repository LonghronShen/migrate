CREATE TABLE "chartered_flight" (flight_no NUMBER(4) PRIMARY KEY
, customer_id NUMBER(6)
, aircraft_no NUMBER(4)
, flight_type VARCHAR2 (12)
, flight_date DATE NOT NULL
, flight_time INTERVAL DAY TO SECOND NOT NULL
, takeoff_at CHAR (3) NOT NULL 
, destination CHAR (3) NOT NULL 
)