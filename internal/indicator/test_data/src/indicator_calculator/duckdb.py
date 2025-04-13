import duckdb
import pandas as pd


def read_parquet_to_df(path: str) -> pd.DataFrame:
    """
    Read a parquet file into a pandas dataframe.
    Args:
        path (str): The path to the parquet file.
    Returns:
        pd.DataFrame: The dataframe containing the data from the parquet file.
    """
    # Use duckdb to read the parquet file
    df = duckdb.query(f"SELECT * FROM '{path}'").to_df()
    return df

def write_df_to_parquet(df: pd.DataFrame, path: str) -> None:
    """
    Write a pandas dataframe to a parquet file.
    Args:
        df (pd.DataFrame): The dataframe to write.
        path (str): The path to the output parquet file.
    """
    # Use duckdb to write the dataframe to a parquet file
    duckdb.from_df(df).write_parquet(path)