#!/usr/bin/env python3

import json
import os

import httpx
from mcp.server.fastmcp import FastMCP


INSTRUCTIONS = """
Bocha is a Chinese search engine for AI. This server provides tools for
searching the web with the Bocha Search API and returning readable summaries.
"""

server = FastMCP("bocha-search-mcp", instructions=INSTRUCTIONS)


def _bocha_api_key() -> str:
    return os.environ.get("BOCHA_API_KEY", "").strip()


def _verify_target():
    return (
        os.environ.get("REQUESTS_CA_BUNDLE")
        or os.environ.get("SSL_CERT_FILE")
        or True
    )


async def _post_json(endpoint: str, payload: dict) -> dict | str:
    api_key = _bocha_api_key()
    if not api_key:
        return "Error: BOCHA_API_KEY environment variable is not set."

    headers = {
        "Authorization": f"Bearer {api_key}",
        "Content-Type": "application/json",
    }

    try:
        async with httpx.AsyncClient(timeout=15.0, verify=_verify_target()) as client:
            response = await client.post(endpoint, headers=headers, json=payload)
            response.raise_for_status()
            return response.json()
    except httpx.HTTPStatusError as exc:
        return f"Bocha API HTTP error: {exc.response.status_code} - {exc.response.text}"
    except httpx.RequestError as exc:
        return f"Error communicating with Bocha API: {exc}"
    except Exception as exc:
        return f"Unexpected error: {exc}"


@server.tool()
async def bocha_web_search(query: str, freshness: str = "noLimit", count: int = 10) -> str:
    """Search the web with Bocha and return text summaries.

    Args:
        query: Search query.
        freshness: Date filter. Supports Bocha values like noLimit, oneDay, oneWeek.
        count: Number of results to return, between 1 and 50.
    """

    payload = {
        "query": query,
        "summary": True,
        "freshness": freshness,
        "count": count,
    }
    response = await _post_json(
        "https://api.bochaai.com/v1/web-search",
        payload,
    )
    if isinstance(response, str):
        return response

    data = response.get("data", {})
    pages = data.get("webPages", {}).get("value", [])
    if not pages:
        return "No results found."

    results = []
    for item in pages:
        results.append(
            "\n".join(
                [
                    f"Title: {item.get('name', '')}",
                    f"URL: {item.get('url', '')}",
                    f"Description: {item.get('summary', '')}",
                    f"Published date: {item.get('datePublished', '')}",
                    f"Site name: {item.get('siteName', '')}",
                ]
            )
        )
    return "\n\n".join(results)


@server.tool()
async def bocha_ai_search(query: str, freshness: str = "noLimit", count: int = 10) -> str:
    """Search with Bocha AI Search and return structured text results.

    Args:
        query: Search query.
        freshness: Date filter. Supports Bocha values like noLimit, oneDay, oneWeek.
        count: Number of results to return, between 1 and 50.
    """

    payload = {
        "query": query,
        "freshness": freshness,
        "count": count,
        "answer": False,
        "stream": False,
    }
    response = await _post_json(
        "https://api.bochaai.com/v1/ai-search",
        payload,
    )
    if isinstance(response, str):
        return response

    results = []
    for message in response.get("messages", []):
        content_type = message.get("content_type")
        content = message.get("content", "")

        if content_type == "webpage":
            try:
                parsed = json.loads(content)
            except json.JSONDecodeError:
                parsed = {}

            for item in parsed.get("value", []):
                results.append(
                    "\n".join(
                        [
                            f"Title: {item.get('name', '')}",
                            f"URL: {item.get('url', '')}",
                            f"Description: {item.get('summary', '')}",
                            f"Published date: {item.get('datePublished', '')}",
                            f"Site name: {item.get('siteName', '')}",
                        ]
                    )
                )
            continue

        if content_type != "image" and content and content != "{}":
            results.append(content)

    return "\n\n".join(results) if results else "No results found."


if __name__ == "__main__":
    server.run(transport="stdio")
