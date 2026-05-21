MAX_WAIT=120
WAIT_INTERVAL=5
ELAPSED=0

while ! kafka-topics --bootstrap-server kafka:9092 --list &>/dev/null; do
    if [ $ELAPSED -ge $MAX_WAIT ]; then
        exit 1
    fi
    sleep $WAIT_INTERVAL
    ELAPSED=$((ELAPSED + WAIT_INTERVAL))
done

echo "Kafka is ready! Creating topics..."

create_topic() {
    local topic=$1
    local partitions=$2
    local replication=$3
    local retention=$4
    local cleanup=$5
    
    echo "Creating topic: $topic"
    
    kafka-topics --bootstrap-server kafka:9092 --create --if-not-exists \
        --topic "$topic" \
        --partitions "$partitions" \
        --replication-factor "$replication" \
        --config "retention.ms=$retention" \
        --config "cleanup.policy=$cleanup" \
        --config "min.insync.replicas=1" \
        --config "compression.type=snappy"
    
    if [ $? -eq 0 ]; then
        echo "Topic '$topic' created/verified successfully"
    else
        echo "Failed to create topic '$topic'"
        return 1
    fi
}

create_topic "account-commands" 3 1 86400000 "delete"
create_topic "transaction-commands" 3 1 86400000 "delete"
create_topic "external-transaction-commands" 3 1 86400000 "delete"
create_topic "dlq" 1 1 604800000 "delete"

kafka-topics --bootstrap-server kafka:9092 --list
kafka-topics --bootstrap-server kafka:9092 --describe
