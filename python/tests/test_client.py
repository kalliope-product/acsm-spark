from acsm_spark import ACSManagerClient, ConnectSessionRequest, ConnectSession
from pyspark.sql import SparkSession, functions as F

client = ACSManagerClient("localhost:15003")

request = ConnectSessionRequest(session_name="test-session")
response = client.GetOrCreateSession(request)
print(response)

spark = SparkSession.builder.remote(
    f"sc://localhost:15003/;session-id={response.session_id}"
).getOrCreate()

spark.range(10).withColumn("test", F.col("id") + 10).show()

spark.stop()
