from pyspark.sql import SparkSession
from pyspark.sql import functions as F
# New spark remote session
spark = SparkSession.builder.remote(
    "sc://localhost:15003/;spark-instance=019b2013-9e63-7521-874f-33d3f215c506"
).getOrCreate()

df = spark.range(10).withColumn("test", F.col("id") + 10)
df.show()